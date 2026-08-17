package handlers

import (
	"context"
	"errors"
	"net/http"
	"scenic-ticket/config"
	"scenic-ticket/ticket"
	"time"

	"github.com/gin-gonic/gin"
)

type LifecycleHandler struct {
	service *ticket.Service
}

func NewLifecycleHandler(service *ticket.Service) *LifecycleHandler {
	return &LifecycleHandler{service: service}
}

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

func (h *LifecycleHandler) SellTicket(c *gin.Context) {
	var request SellTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	visitDate, err := config.ParseDate(request.VisitDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，请使用 YYYY-MM-DD"})
		return
	}
	sellerID, idOK := c.Get("user_id")
	sellerName, nameOK := c.Get("name")
	id, idTypeOK := sellerID.(uint)
	name, nameTypeOK := sellerName.(string)
	if !idOK || !nameOK || !idTypeOK || !nameTypeOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户上下文无效"})
		return
	}
	result, err := h.service.Sell(c.Request.Context(), ticket.SellInput{
		TicketTypeID: request.TicketTypeID, BuyerName: request.BuyerName, Phone: request.Phone,
		VisitDate: visitDate, TimeSlotID: request.TimeSlotID, SellerID: id, SellerName: name,
	})
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, SellTicketResponse{TicketNo: result.TicketNo, Message: result.Message})
}

type RefundTicketRequest struct {
	TicketNo string `json:"ticket_no" binding:"required"`
}

func (h *LifecycleHandler) RefundTicket(c *gin.Context) {
	var request RefundTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.service.Refund(c.Request.Context(), request.TicketNo); err != nil {
		writeLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "退票成功"})
}

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

func (h *LifecycleHandler) CheckIn(c *gin.Context) {
	var request CheckInRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	result, err := h.service.CheckIn(c.Request.Context(), ticket.CheckInput{TicketNo: request.TicketNo, GateID: request.GateID})
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, CheckInResponse{
		TicketNo: result.TicketNo, CheckTime: result.CheckTime, GateName: result.GateName, Message: result.Message,
	})
}

type CheckOutRequest struct {
	TicketNo string `json:"ticket_no" binding:"required"`
	GateID   uint   `json:"gate_id" binding:"required"`
}

func (h *LifecycleHandler) CheckOut(c *gin.Context) {
	var request CheckOutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	result, err := h.service.CheckOut(c.Request.Context(), ticket.CheckInput{TicketNo: request.TicketNo, GateID: request.GateID})
	if err != nil {
		writeLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ticket_no": result.TicketNo, "check_time": result.CheckTime, "gate_name": result.GateName, "message": result.Message,
	})
}

func writeLifecycleError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ticket.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, ticket.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ticket.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status = http.StatusRequestTimeout
	}
	c.JSON(status, gin.H{"error": ticket.PublicMessage(err)})
}
