package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"scenic-ticket/handlers"
	"scenic-ticket/ticket"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type errorRepository struct {
	err error
}

func (r errorRepository) WithTx(ctx context.Context, _ func(ticket.TxRepository) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.err
}

type testClock struct{}

func (testClock) Now() time.Time {
	return time.Date(2026, 8, 17, 10, 30, 0, 0, time.UTC)
}

type testNumbers struct{}

func (testNumbers) Generate(time.Time) (string, error) {
	return "20260817000000000001", nil
}

func TestLifecycleHandlerMapsServiceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "not found", err: &ticket.Error{Kind: ticket.ErrNotFound, Message: "票不存在"}, status: http.StatusNotFound},
		{name: "conflict", err: &ticket.Error{Kind: ticket.ErrConflict, Message: "该票已退票"}, status: http.StatusConflict},
		{name: "storage", err: &ticket.Error{Kind: ticket.ErrStorage, Message: "保存失败"}, status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ticket.NewService(errorRepository{err: tt.err}, testClock{}, testNumbers{})
			handler := handlers.NewLifecycleHandler(service)
			router := gin.New()
			router.POST("/tickets/refund", handler.RefundTicket)
			request := httptest.NewRequest(http.MethodPost, "/tickets/refund", bytes.NewBufferString(`{"ticket_no":"T-1"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.status, response.Body.String())
			}
		})
	}
}

func TestGetCheckRecordsRejectsInvalidDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/check-records", handlers.GetCheckRecords)

	request := httptest.NewRequest(http.MethodGet, "/check-records?date=not-a-date", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}
