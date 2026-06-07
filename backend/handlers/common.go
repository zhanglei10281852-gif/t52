package handlers

import (
	"net/http"
	"scenic-ticket/database"
	"scenic-ticket/models"

	"github.com/gin-gonic/gin"
)

func GetTicketTypes(c *gin.Context) {
	var ticketTypes []models.TicketType
	database.DB.Where("status = ?", 1).Find(&ticketTypes)
	c.JSON(http.StatusOK, ticketTypes)
}

func CreateTicketType(c *gin.Context) {
	var ticketType models.TicketType
	if err := c.ShouldBindJSON(&ticketType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	database.DB.Create(&ticketType)
	c.JSON(http.StatusOK, ticketType)
}

func GetGates(c *gin.Context) {
	var gates []models.Gate
	database.DB.Where("status = ?", 1).Find(&gates)
	c.JSON(http.StatusOK, gates)
}

func GetUsers(c *gin.Context) {
	role := c.Query("role")
	var users []models.User
	query := database.DB
	if role != "" {
		query = query.Where("role = ?", role)
	}
	query.Select("id, username, name, role, created_at").Find(&users)
	c.JSON(http.StatusOK, users)
}
