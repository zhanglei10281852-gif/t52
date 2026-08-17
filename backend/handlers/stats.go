package handlers

import (
	"net/http"
	"scenic-ticket/config"
	"scenic-ticket/database"
	"scenic-ticket/models"
	"time"

	"github.com/gin-gonic/gin"
)

type DashboardData struct {
	InParkCount   int            `json:"in_park_count"`
	MaxCapacity   int            `json:"max_capacity"`
	TodayCheckIn  int            `json:"today_check_in"`
	TodayCheckOut int            `json:"today_check_out"`
	TodaySold     int            `json:"today_sold"`
	SlotInfo      []SlotInfoItem `json:"slot_info"`
}

type SlotInfoItem struct {
	SlotID      int    `json:"slot_id"`
	SlotName    string `json:"slot_name"`
	MaxCapacity int    `json:"max_capacity"`
	SoldCount   int    `json:"sold_count"`
	Remaining   int    `json:"remaining"`
}

func GetDashboard(c *gin.Context) {
	today := config.DateKey(time.Now())

	var dailyCapacity models.DailyCapacity
	var dailyExists = true
	err := database.DB.Where("date = ?", today).First(&dailyCapacity).Error
	if err != nil {
		dailyExists = false
		dailyCapacity.MaxCapacity = config.MaxDailyCapacity
	}

	var checkedInCount int
	if dailyExists {
		checkedInCount = dailyCapacity.CheckedInCount
	}

	checkedOutCount := getTodayCheckedOutCountDB(today)
	inParkCount := checkedInCount - checkedOutCount
	if inParkCount < 0 {
		inParkCount = 0
	}

	var soldCount int
	if dailyExists {
		soldCount = dailyCapacity.SoldCount
	}

	slotInfo := make([]SlotInfoItem, 0)
	for _, slot := range config.TimeSlots {
		var slotCapacity models.SlotCapacity
		maxCapacity := config.DefaultSlotCapacity
		remaining := config.DefaultSlotCapacity
		sold := 0
		err := database.DB.Where("date = ? AND time_slot_id = ?", today, slot.ID).First(&slotCapacity).Error
		if err == nil {
			maxCapacity = slotCapacity.MaxCapacity
			sold = slotCapacity.SoldCount
			remaining = slotCapacity.MaxCapacity - slotCapacity.SoldCount
			if remaining < 0 {
				remaining = 0
			}
		}

		slotInfo = append(slotInfo, SlotInfoItem{
			SlotID:      slot.ID,
			SlotName:    slot.Name,
			MaxCapacity: maxCapacity,
			SoldCount:   sold,
			Remaining:   remaining,
		})
	}

	data := DashboardData{
		InParkCount:   inParkCount,
		MaxCapacity:   dailyCapacity.MaxCapacity,
		TodayCheckIn:  checkedInCount,
		TodayCheckOut: checkedOutCount,
		TodaySold:     soldCount,
		SlotInfo:      slotInfo,
	}

	c.JSON(http.StatusOK, data)
}

type GateStatsItem struct {
	GateID   uint   `json:"gate_id"`
	GateName string `json:"gate_name"`
	InCount  int    `json:"in_count"`
	OutCount int    `json:"out_count"`
	Total    int    `json:"total"`
}

func GetGateStats(c *gin.Context) {
	today := config.DateKey(time.Now())
	startDate, endDate, ok := parseDateRange(c, today, today)
	if !ok {
		return
	}
	startTime, _ := config.DayRange(startDate)
	_, endTime := config.DayRange(endDate.AddDate(0, 0, -1))

	var records []models.CheckRecord
	database.DB.Where("check_time >= ? AND check_time < ?", startTime, endTime).Find(&records)

	gateMap := make(map[uint]*GateStatsItem)
	for _, r := range records {
		if _, ok := gateMap[r.GateID]; !ok {
			gateMap[r.GateID] = &GateStatsItem{
				GateID:   r.GateID,
				GateName: r.GateName,
			}
		}
		if r.CheckType == models.CheckTypeIn {
			gateMap[r.GateID].InCount++
		} else {
			gateMap[r.GateID].OutCount++
		}
		gateMap[r.GateID].Total++
	}

	result := make([]GateStatsItem, 0)
	for _, v := range gateMap {
		result = append(result, *v)
	}

	c.JSON(http.StatusOK, result)
}

type HourlyStatsItem struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

func GetHourlyStats(c *gin.Context) {
	dateStr := c.Query("date")
	date := config.DateKey(time.Now())
	if dateStr != "" {
		parsed, err := config.ParseDate(dateStr)
		if err != nil {
			writeDateError(c, "日期格式错误，请使用 YYYY-MM-DD")
			return
		}
		date = parsed
	}

	startTime, endTime := config.DayRange(date)

	var records []models.CheckRecord
	database.DB.Where("check_type = ? AND check_time >= ? AND check_time < ?",
		models.CheckTypeIn, startTime, endTime).Find(&records)

	hourMap := make(map[int]int)
	for i := 0; i < 24; i++ {
		hourMap[i] = 0
	}
	for _, r := range records {
		hour := r.CheckTime.In(config.BusinessLocation).Hour()
		hourMap[hour]++
	}

	result := make([]HourlyStatsItem, 0)
	for i := 0; i < 24; i++ {
		result = append(result, HourlyStatsItem{
			Hour:  i,
			Count: hourMap[i],
		})
	}

	c.JSON(http.StatusOK, result)
}

type DailyStatsItem struct {
	Date         string  `json:"date"`
	CheckInCount int     `json:"check_in_count"`
	SoldCount    int     `json:"sold_count"`
	TotalRevenue float64 `json:"total_revenue"`
}

func GetDailyStats(c *gin.Context) {
	today := config.DateKey(time.Now())
	startTime, endTime, ok := parseDateRange(c, today.AddDate(0, 0, -29), today)
	if !ok {
		return
	}

	var dailyCapacities []models.DailyCapacity
	database.DB.Where("date >= ? AND date < ?", startTime, endTime).
		Order("date asc").Find(&dailyCapacities)

	capMap := make(map[string]*models.DailyCapacity)
	for i := range dailyCapacities {
		d := dailyCapacities[i]
		capMap[d.Date.In(time.UTC).Format(config.DateLayout)] = &d
	}

	type TicketSum struct {
		VisitDate    time.Time
		TotalRevenue float64
		SoldCount    int
	}
	var ticketSums []TicketSum
	database.DB.Model(&models.Ticket{}).
		Select("visit_date, SUM(price) as total_revenue, COUNT(*) as sold_count").
		Where("visit_date >= ? AND visit_date < ? AND status != ?",
			startTime, endTime, models.TicketStatusRefunded).
		Group("visit_date").
		Scan(&ticketSums)

	revenueMap := make(map[string]float64)
	soldMap := make(map[string]int)
	for _, ts := range ticketSums {
		key := ts.VisitDate.In(time.UTC).Format(config.DateLayout)
		revenueMap[key] = ts.TotalRevenue
		soldMap[key] = ts.SoldCount
	}

	result := make([]DailyStatsItem, 0)
	for d := startTime; d.Before(endTime); d = d.AddDate(0, 0, 1) {
		key := d.Format(config.DateLayout)
		item := DailyStatsItem{Date: key}
		if cap, ok := capMap[key]; ok {
			item.CheckInCount = cap.CheckedInCount
		}
		if rev, ok := revenueMap[key]; ok {
			item.TotalRevenue = rev
		}
		if sc, ok := soldMap[key]; ok {
			item.SoldCount = sc
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

type TicketTypeRevenueItem struct {
	TicketTypeID   uint    `json:"ticket_type_id"`
	TicketTypeName string  `json:"ticket_type_name"`
	Count          int     `json:"count"`
	Revenue        float64 `json:"revenue"`
}

func GetTicketTypeStats(c *gin.Context) {
	today := config.DateKey(time.Now())
	startTime, endTime, ok := parseDateRange(c, today.AddDate(0, 0, -29), today)
	if !ok {
		return
	}

	type Result struct {
		TicketTypeID   uint
		TicketTypeName string
		Count          int64
		Revenue        float64
	}
	var results []Result

	database.DB.Table("tickets").
		Select("tickets.ticket_type_id, ticket_types.name as ticket_type_name, COUNT(*) as count, SUM(tickets.price) as revenue").
		Joins("LEFT JOIN ticket_types ON tickets.ticket_type_id = ticket_types.id").
		Where("tickets.visit_date >= ? AND tickets.visit_date < ? AND tickets.status != ?",
			startTime, endTime, models.TicketStatusRefunded).
		Group("tickets.ticket_type_id, ticket_types.name").
		Scan(&results)

	finalResult := make([]TicketTypeRevenueItem, 0)
	for _, r := range results {
		finalResult = append(finalResult, TicketTypeRevenueItem{
			TicketTypeID:   r.TicketTypeID,
			TicketTypeName: r.TicketTypeName,
			Count:          int(r.Count),
			Revenue:        r.Revenue,
		})
	}

	c.JSON(http.StatusOK, finalResult)
}

type SlotHeatmapItem struct {
	Date     string `json:"date"`
	SlotID   int    `json:"slot_id"`
	SlotName string `json:"slot_name"`
	Count    int    `json:"count"`
}

func GetSlotHeatmap(c *gin.Context) {
	today := config.DateKey(time.Now())
	startTime, endTime, ok := parseDateRange(c, today.AddDate(0, 0, -6), today)
	if !ok {
		return
	}

	var slotCapacities []models.SlotCapacity
	database.DB.Where("date >= ? AND date < ?", startTime, endTime).
		Order("date asc, time_slot_id asc").Find(&slotCapacities)

	result := make([]SlotHeatmapItem, 0)
	for d := startTime; d.Before(endTime); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format(config.DateLayout)
		for _, slot := range config.TimeSlots {
			count := 0
			for _, sc := range slotCapacities {
				if sc.Date.In(time.UTC).Format(config.DateLayout) == dateStr && sc.TimeSlotID == slot.ID {
					count = sc.CheckedInCount
					break
				}
			}
			result = append(result, SlotHeatmapItem{
				Date:     dateStr,
				SlotID:   slot.ID,
				SlotName: slot.Name,
				Count:    count,
			})
		}
	}

	c.JSON(http.StatusOK, result)
}

func parseDateRange(c *gin.Context, defaultStart, defaultEnd time.Time) (time.Time, time.Time, bool) {
	start := defaultStart
	if value := c.Query("start_date"); value != "" {
		parsed, err := config.ParseDate(value)
		if err != nil {
			writeDateError(c, "开始日期格式错误，请使用 YYYY-MM-DD")
			return time.Time{}, time.Time{}, false
		}
		start = parsed
	}

	end := defaultEnd
	if value := c.Query("end_date"); value != "" {
		parsed, err := config.ParseDate(value)
		if err != nil {
			writeDateError(c, "结束日期格式错误，请使用 YYYY-MM-DD")
			return time.Time{}, time.Time{}, false
		}
		end = parsed
	}
	if end.Before(start) {
		writeDateError(c, "结束日期不能早于开始日期")
		return time.Time{}, time.Time{}, false
	}
	return start, end.AddDate(0, 0, 1), true
}

func writeDateError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}
