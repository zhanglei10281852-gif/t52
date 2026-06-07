package handlers

import (
	"net/http"
	"scenic-ticket/config"
	"scenic-ticket/database"
	"scenic-ticket/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CheckInRequest struct {
	TicketNo string `json:"ticket_no" binding:"required"`
	GateID   uint   `json:"gate_id" binding:"required"`
}

type CheckInResponse struct {
	TicketNo  string    `json:"ticket_no"`
	CheckTime time.Time `json:"check_time"`
	GateName  string    `json:"gate_name"`
	Message   string    `json:"message"`
}

func CheckIn(c *gin.Context) {
	var req CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	tx := database.DB.Begin()

	var ticket models.Ticket
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("ticket_no = ?", req.TicketNo).First(&ticket).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "票不存在"})
		return
	}

	if ticket.Status == models.TicketStatusRefunded {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "该票已退票"})
		return
	}

	if ticket.Status == models.TicketStatusUsed {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "该票已使用"})
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	visitDay := ticket.VisitDate.Truncate(24 * time.Hour)
	if !visitDay.Equal(today) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "该票不是今天的票"})
		return
	}

	slotInfo := config.GetTimeSlotInfo(ticket.TimeSlotID)
	if slotInfo != nil {
		now := time.Now()
		slotStartTime := time.Date(today.Year(), today.Month(), today.Day(), slotInfo.StartHour, 0, 0, 0, today.Location())
		allowedEntryTime := slotStartTime.Add(-30 * time.Minute)

		if now.Before(allowedEntryTime) {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "未到入园时间，请提前30分钟内入园"})
			return
		}
	}

	var gate models.Gate
	if err := tx.First(&gate, req.GateID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "闸机不存在"})
		return
	}

	dailyCapacity, err := getOrCreateDailyCapacity(tx, today)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日容量信息失败"})
		return
	}

	currentInPark := dailyCapacity.CheckedInCount - getTodayCheckedOutCount(tx, today)
	if currentInPark >= dailyCapacity.MaxCapacity {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "景区满员"})
		return
	}

	now := time.Now()
	gateIDPtr := &gate.ID
	ticket.Status = models.TicketStatusUsed
	ticket.CheckInGateID = gateIDPtr
	ticket.CheckInTime = &now
	if err := tx.Save(&ticket).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新票状态失败"})
		return
	}

	checkRecord := models.CheckRecord{
		TicketID:   ticket.ID,
		TicketNo:   ticket.TicketNo,
		GateID:     gate.ID,
		GateName:   gate.Name,
		CheckType:  models.CheckTypeIn,
		CheckTime:  now,
		TimeSlotID: ticket.TimeSlotID,
	}
	if err := tx.Create(&checkRecord).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建核销记录失败"})
		return
	}

	dailyCapacity.CheckedInCount++
	tx.Save(dailyCapacity)

	var slotCapacity models.SlotCapacity
	if err := tx.Where("date = ? AND time_slot_id = ?", today, ticket.TimeSlotID).First(&slotCapacity).Error; err == nil {
		slotCapacity.CheckedInCount++
		tx.Save(&slotCapacity)
	}

	tx.Commit()

	c.JSON(http.StatusOK, CheckInResponse{
		TicketNo:  ticket.TicketNo,
		CheckTime: now,
		GateName:  gate.Name,
		Message:   "入园成功",
	})
}

func getTodayCheckedOutCount(tx *gorm.DB, date time.Time) int {
	var count int64
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)
	tx.Model(&models.CheckRecord{}).
		Where("check_type = ? AND check_time >= ? AND check_time < ?",
			models.CheckTypeOut, startOfDay, endOfDay).
		Count(&count)
	return int(count)
}

type CheckOutRequest struct {
	TicketNo string `json:"ticket_no" binding:"required"`
	GateID   uint   `json:"gate_id" binding:"required"`
}

func CheckOut(c *gin.Context) {
	var req CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	tx := database.DB.Begin()

	var ticket models.Ticket
	if err := tx.Where("ticket_no = ?", req.TicketNo).First(&ticket).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "票不存在"})
		return
	}

	if ticket.Status != models.TicketStatusUsed {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "该票未入园"})
		return
	}

	if ticket.CheckOutTime != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "该票已出园"})
		return
	}

	var gate models.Gate
	if err := tx.First(&gate, req.GateID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "闸机不存在"})
		return
	}

	now := time.Now()
	gateIDPtr := &gate.ID
	ticket.CheckOutGateID = gateIDPtr
	ticket.CheckOutTime = &now
	if err := tx.Save(&ticket).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新票状态失败"})
		return
	}

	checkRecord := models.CheckRecord{
		TicketID:   ticket.ID,
		TicketNo:   ticket.TicketNo,
		GateID:     gate.ID,
		GateName:   gate.Name,
		CheckType:  models.CheckTypeOut,
		CheckTime:  now,
		TimeSlotID: ticket.TimeSlotID,
	}
	if err := tx.Create(&checkRecord).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建出园记录失败"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"ticket_no":  ticket.TicketNo,
		"check_time": now,
		"gate_name":  gate.Name,
		"message":    "出园成功",
	})
}

type InParkResponse struct {
	Count        int `json:"count"`
	MaxCapacity  int `json:"max_capacity"`
	Percentage   float64 `json:"percentage"`
}

func GetInParkCount(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	var dailyCapacity models.DailyCapacity
	err := database.DB.Where("date = ?", today).First(&dailyCapacity).Error
	if err != nil {
		c.JSON(http.StatusOK, InParkResponse{
			Count:       0,
			MaxCapacity: config.MaxDailyCapacity,
			Percentage:  0,
		})
		return
	}

	checkedOutCount := getTodayCheckedOutCountDB(dailyCapacity)
	inParkCount := dailyCapacity.CheckedInCount - checkedOutCount
	if inParkCount < 0 {
		inParkCount = 0
	}

	percentage := float64(inParkCount) / float64(dailyCapacity.MaxCapacity) * 100

	c.JSON(http.StatusOK, InParkResponse{
		Count:       inParkCount,
		MaxCapacity: dailyCapacity.MaxCapacity,
		Percentage:  percentage,
	})
}

func getTodayCheckedOutCountDB(dailyCapacity models.DailyCapacity) int {
	var count int64
	date := dailyCapacity.Date
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)
	database.DB.Model(&models.CheckRecord{}).
		Where("check_type = ? AND check_time >= ? AND check_time < ?",
			models.CheckTypeOut, startOfDay, endOfDay).
		Count(&count)
	return int(count)
}

func GetCheckRecords(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	checkType := c.Query("check_type")
	date := c.Query("date")

	var records []models.CheckRecord
	var total int64

	query := database.DB.Model(&models.CheckRecord{})

	if checkType != "" {
		query = query.Where("check_type = ?", checkType)
	}
	if date != "" {
		d, _ := time.Parse("2006-01-02", date)
		start := d.Truncate(24 * time.Hour)
		end := start.Add(24 * time.Hour)
		query = query.Where("check_time >= ? AND check_time < ?", start, end)
	}

	query.Count(&total)

	offset := (parseInt(page) - 1) * parseInt(pageSize)
	query.Order("check_time desc").Offset(offset).Limit(parseInt(pageSize)).Find(&records)

	c.JSON(http.StatusOK, gin.H{
		"list":  records,
		"total": total,
		"page":  parseInt(page),
		"size":  parseInt(pageSize),
	})
}
