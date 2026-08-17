package handlers

import (
	"net/http"
	"scenic-ticket/config"
	"scenic-ticket/database"
	"scenic-ticket/models"
	"time"

	"github.com/gin-gonic/gin"
)

type InParkResponse struct {
	Count       int     `json:"count"`
	MaxCapacity int     `json:"max_capacity"`
	Percentage  float64 `json:"percentage"`
}

func GetInParkCount(c *gin.Context) {
	today := config.DateKey(time.Now())
	var dailyCapacity models.DailyCapacity
	if err := database.DB.Where("date = ?", today).First(&dailyCapacity).Error; err != nil {
		c.JSON(http.StatusOK, InParkResponse{Count: 0, MaxCapacity: config.MaxDailyCapacity, Percentage: 0})
		return
	}
	checkedOutCount := getTodayCheckedOutCountDB(today)
	inParkCount := dailyCapacity.CheckedInCount - checkedOutCount
	if inParkCount < 0 {
		inParkCount = 0
	}
	percentage := float64(inParkCount) / float64(dailyCapacity.MaxCapacity) * 100
	c.JSON(http.StatusOK, InParkResponse{
		Count: inParkCount, MaxCapacity: dailyCapacity.MaxCapacity, Percentage: percentage,
	})
}

func getTodayCheckedOutCountDB(date time.Time) int {
	var count int64
	startOfDay, endOfDay := config.DayRange(date)
	database.DB.Model(&models.CheckRecord{}).
		Where("check_type = ? AND check_time >= ? AND check_time < ?", models.CheckTypeOut, startOfDay, endOfDay).
		Count(&count)
	return int(count)
}

func GetCheckRecords(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	checkType := c.Query("check_type")
	date := c.Query("date")
	var start, end time.Time
	if date != "" {
		parsed, err := config.ParseDate(date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，请使用 YYYY-MM-DD"})
			return
		}
		start, end = config.DayRange(parsed)
	}

	var records []models.CheckRecord
	var total int64
	query := database.DB.Model(&models.CheckRecord{})
	if checkType != "" {
		query = query.Where("check_type = ?", checkType)
	}
	if date != "" {
		query = query.Where("check_time >= ? AND check_time < ?", start, end)
	}
	query.Count(&total)
	offset := (parseInt(page) - 1) * parseInt(pageSize)
	query.Order("check_time desc").Offset(offset).Limit(parseInt(pageSize)).Find(&records)
	c.JSON(http.StatusOK, gin.H{
		"list": records, "total": total, "page": parseInt(page), "size": parseInt(pageSize),
	})
}
