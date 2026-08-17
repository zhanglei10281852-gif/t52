package handlers

import (
	"fmt"
	"net/http"
	"scenic-ticket/database"
	"scenic-ticket/models"

	"github.com/gin-gonic/gin"
)

func GetTicket(c *gin.Context) {
	ticketNo := c.Param("ticket_no")
	var value models.Ticket
	if err := database.DB.Preload("TicketType").Preload("CheckInGate").Preload("CheckOutGate").
		Where("ticket_no = ?", ticketNo).First(&value).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "票不存在"})
		return
	}
	c.JSON(http.StatusOK, value)
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
		"list": tickets, "total": total, "page": parseInt(page), "size": parseInt(pageSize),
	})
}

func parseInt(value string) int {
	var result int
	fmt.Sscanf(value, "%d", &result)
	return result
}
