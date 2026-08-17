package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"scenic-ticket/handlers"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStatsHandlersRejectInvalidDatesBeforeQuerying(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		path string
	}{
		{path: "/stats/gates?start_date=invalid"},
		{path: "/stats/hourly?date=invalid"},
		{path: "/stats/daily?end_date=invalid"},
		{path: "/stats/ticket-types?start_date=invalid"},
		{path: "/stats/slot-heatmap?end_date=invalid"},
		{path: "/stats/daily?start_date=2026-08-18&end_date=2026-08-17"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			router := gin.New()
			router.GET("/stats/gates", handlers.GetGateStats)
			router.GET("/stats/hourly", handlers.GetHourlyStats)
			router.GET("/stats/daily", handlers.GetDailyStats)
			router.GET("/stats/ticket-types", handlers.GetTicketTypeStats)
			router.GET("/stats/slot-heatmap", handlers.GetSlotHeatmap)

			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}
