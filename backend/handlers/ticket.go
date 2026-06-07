package handlers

import (
	"fmt"
	"net/http"
	"scenic-ticket/config"
	"scenic-ticket/database"
	"scenic-ticket/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SellTicketRequest struct {
	TicketTypeID uint   `json:"ticket_type_id" binding:"required"`
	BuyerName    string `json:"buyer_name" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	VisitDate    string `json:"visit_date" binding:"required"`
	TimeSlotID   int    `json:"time_slot_id" binding:"required"`
}

type SellTicketResponse struct {
	TicketNo string `json:"ticket_no"`
	Message  string `json:"message"`
}

func SellTicket(c *gin.Context) {
	var req SellTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	visitDate, err := time.Parse("2006-01-02", req.VisitDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，请使用 YYYY-MM-DD"})
		return
	}
	visitDate = visitDate.Truncate(24 * time.Hour)

	slotInfo := config.GetTimeSlotInfo(req.TimeSlotID)
	if slotInfo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的时段"})
		return
	}

	sellerID, _ := c.Get("user_id")
	sellerName, _ := c.Get("name")

	var ticketType models.TicketType
	if err := database.DB.First(&ticketType, req.TicketTypeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "票型不存在"})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var phoneCount int64
	tx.Model(&models.Ticket{}).Where(
		"phone = ? AND visit_date = ? AND status != ?",
		req.Phone, visitDate, models.TicketStatusRefunded,
	).Count(&phoneCount)
	if phoneCount >= int64(config.MaxTicketsPerPhone) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("同一手机号每天限购%d张票", config.MaxTicketsPerPhone)})
		return
	}

	dailyCapacity, err := getOrCreateDailyCapacity(tx, visitDate)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日容量信息失败"})
		return
	}

	maxSell := int(float64(dailyCapacity.MaxCapacity) * config.CapacityOverSell)
	if dailyCapacity.SoldCount >= maxSell {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "当日票已售罄"})
		return
	}

	slotCapacity, err := getOrCreateSlotCapacity(tx, visitDate, req.TimeSlotID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取时段容量信息失败"})
		return
	}
	if slotCapacity.SoldCount >= slotCapacity.MaxCapacity {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "该时段票已售罄"})
		return
	}

	ticketNo, err := generateTicketNo(tx, visitDate)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成票号失败"})
		return
	}

	ticket := models.Ticket{
		TicketNo:     ticketNo,
		TicketTypeID: req.TicketTypeID,
		BuyerName:    req.BuyerName,
		Phone:        req.Phone,
		VisitDate:    visitDate,
		TimeSlotID:   req.TimeSlotID,
		TimeSlotName: slotInfo.Name,
		Price:        ticketType.Price,
		Status:       models.TicketStatusUnused,
		SellerID:     sellerID.(uint),
		SellerName:   sellerName.(string),
	}
	if err := tx.Create(&ticket).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建票务失败"})
		return
	}

	dailyCapacity.SoldCount++
	tx.Save(dailyCapacity)

	slotCapacity.SoldCount++
	tx.Save(slotCapacity)

	tx.Commit()

	ticket.TicketType = ticketType
	c.JSON(http.StatusOK, SellTicketResponse{
		TicketNo: ticketNo,
		Message:  "售票成功",
	})
}

func generateTicketNo(tx *gorm.DB, date time.Time) (string, error) {
	dateStr := date.Format("20060102")
	prefix := dateStr

	var count int64
	tx.Model(&models.Ticket{}).Where("ticket_no LIKE ?", prefix+"%").Count(&count)

	seq := count + 1
	ticketNo := fmt.Sprintf("%s%06d", dateStr, seq)

	return ticketNo, nil
}

func getOrCreateDailyCapacity(tx *gorm.DB, date time.Time) (*models.DailyCapacity, error) {
	var dailyCapacity models.DailyCapacity
	err := tx.Where("date = ?", date).First(&dailyCapacity).Error
	if err == gorm.ErrRecordNotFound {
		dailyCapacity = models.DailyCapacity{
			Date:        date,
			MaxCapacity: config.MaxDailyCapacity,
			SoldCount:   0,
		}
		if err := tx.Create(&dailyCapacity).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &dailyCapacity, nil
}

func getOrCreateSlotCapacity(tx *gorm.DB, date time.Time, slotID int) (*models.SlotCapacity, error) {
	var slotCapacity models.SlotCapacity
	err := tx.Where("date = ? AND time_slot_id = ?", date, slotID).First(&slotCapacity).Error
	if err == gorm.ErrRecordNotFound {
		slotCapacity = models.SlotCapacity{
			Date:        date,
			TimeSlotID:  slotID,
			MaxCapacity: config.DefaultSlotCapacity,
			SoldCount:   0,
		}
		if err := tx.Create(&slotCapacity).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &slotCapacity, nil
}

type RefundTicketRequest struct {
	TicketNo string `json:"ticket_no" binding:"required"`
}

func RefundTicket(c *gin.Context) {
	var req RefundTicketRequest
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

	if ticket.Status == models.TicketStatusRefunded {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "该票已退票"})
		return
	}

	if ticket.Status == models.TicketStatusUsed {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "已使用的票不能退票"})
		return
	}

	ticket.Status = models.TicketStatusRefunded
	if err := tx.Save(&ticket).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "退票失败"})
		return
	}

	var dailyCapacity models.DailyCapacity
	if err := tx.Where("date = ?", ticket.VisitDate).First(&dailyCapacity).Error; err == nil {
		if dailyCapacity.SoldCount > 0 {
			dailyCapacity.SoldCount--
			tx.Save(&dailyCapacity)
		}
	}

	var slotCapacity models.SlotCapacity
	if err := tx.Where("date = ? AND time_slot_id = ?", ticket.VisitDate, ticket.TimeSlotID).First(&slotCapacity).Error; err == nil {
		if slotCapacity.SoldCount > 0 {
			slotCapacity.SoldCount--
			tx.Save(&slotCapacity)
		}
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "退票成功"})
}

func GetTicket(c *gin.Context) {
	ticketNo := c.Param("ticket_no")

	var ticket models.Ticket
	if err := database.DB.Preload("TicketType").Preload("CheckInGate").Preload("CheckOutGate").
		Where("ticket_no = ?", ticketNo).First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "票不存在"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

func SearchTickets(c *gin.Context) {
	phone := c.Query("phone")
	ticketNo := c.Query("ticket_no")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	var tickets []models.Ticket
	var total int64

	query := database.DB.Model(&models.Ticket{}).Preload("TicketType")

	if phone != "" {
		query = query.Where("phone = ?", phone)
	}
	if ticketNo != "" {
		query = query.Where("ticket_no LIKE ?", "%"+ticketNo+"%")
	}

	role, _ := c.Get("role")
	if role == "seller" {
		sellerID, _ := c.Get("user_id")
		query = query.Where("seller_id = ?", sellerID)
	}

	query.Count(&total)

	offset := (parseInt(page) - 1) * parseInt(pageSize)
	query.Order("created_at desc").Offset(offset).Limit(parseInt(pageSize)).Find(&tickets)

	c.JSON(http.StatusOK, gin.H{
		"list":  tickets,
		"total": total,
		"page":  parseInt(page),
		"size":  parseInt(pageSize),
	})
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
