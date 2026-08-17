package ticket_test

import (
	"context"
	"errors"
	"fmt"
	"scenic-ticket/config"
	"scenic-ticket/database"
	"scenic-ticket/models"
	"scenic-ticket/ticket"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var lifecycleModels = []any{
	&models.TicketType{},
	&models.Gate{},
	&models.Ticket{},
	&models.CheckRecord{},
	&models.DailyCapacity{},
	&models.SlotCapacity{},
	&models.PurchaseQuota{},
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequentialNumbers struct {
	mu   sync.Mutex
	next int
}

func (g *sequentialNumbers) Generate(date time.Time) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return fmt.Sprintf("%s%012d", date.Format("20060102"), g.next), nil
}

type fixture struct {
	service    *ticket.Service
	db         *gorm.DB
	now        time.Time
	visitDate  time.Time
	ticketType models.TicketType
	gate       models.Gate
}

func newFixture(t *testing.T) *fixture {
	return newFixtureAt(
		t,
		time.Date(2026, 8, 17, 10, 30, 0, 0, time.UTC),
		time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC),
	)
}

func newFixtureAt(t *testing.T, now, visitDate time.Time) *fixture {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_foreign_keys=1"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(lifecycleModels...); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	ticketType := models.TicketType{Name: "成人票", Price: 120, Status: 1}
	gate := models.Gate{Name: "东门", Location: "东门", Status: 1}
	if err := db.Create(&ticketType).Error; err != nil {
		t.Fatalf("create ticket type: %v", err)
	}
	if err := db.Create(&gate).Error; err != nil {
		t.Fatalf("create gate: %v", err)
	}
	return &fixture{
		service:    ticket.NewService(database.NewTicketRepository(db), fixedClock{now: now}, &sequentialNumbers{}),
		db:         db,
		now:        now,
		visitDate:  visitDate,
		ticketType: ticketType,
		gate:       gate,
	}
}

func TestCheckInUsesBusinessDayAndSlotBoundary(t *testing.T) {
	visitDate, err := config.ParseDate("2026-08-17")
	if err != nil {
		t.Fatalf("parse visit date: %v", err)
	}
	f := newFixtureAt(t, time.Date(2026, 8, 16, 23, 30, 0, 0, time.UTC), visitDate)
	input := f.sellInput("13800000000")
	input.TimeSlotID = 1
	sold, err := f.service.Sell(context.Background(), input)
	if err != nil {
		t.Fatalf("sell ticket: %v", err)
	}
	if _, err := f.service.CheckIn(context.Background(), ticket.CheckInput{TicketNo: sold.TicketNo, GateID: f.gate.ID}); err != nil {
		t.Fatalf("check in at business slot boundary: %v", err)
	}
}

func (f *fixture) sellInput(phone string) ticket.SellInput {
	return ticket.SellInput{
		TicketTypeID: f.ticketType.ID,
		BuyerName:    "测试游客",
		Phone:        phone,
		VisitDate:    f.visitDate,
		TimeSlotID:   2,
		SellerID:     7,
		SellerName:   "售票员",
	}
}

func TestTicketLifecycle(t *testing.T) {
	f := newFixture(t)
	sold, err := f.service.Sell(context.Background(), f.sellInput("13800000001"))
	if err != nil {
		t.Fatalf("sell ticket: %v", err)
	}

	checkedIn, err := f.service.CheckIn(context.Background(), ticket.CheckInput{TicketNo: sold.TicketNo, GateID: f.gate.ID})
	if err != nil {
		t.Fatalf("check in: %v", err)
	}
	if checkedIn.GateName != f.gate.Name || checkedIn.CheckTime != f.now {
		t.Fatalf("unexpected check-in result: %+v", checkedIn)
	}
	if _, err := f.service.CheckIn(context.Background(), ticket.CheckInput{TicketNo: sold.TicketNo, GateID: f.gate.ID}); !errors.Is(err, ticket.ErrConflict) {
		t.Fatalf("duplicate check-in error = %v, want conflict", err)
	}

	checkedOut, err := f.service.CheckOut(context.Background(), ticket.CheckInput{TicketNo: sold.TicketNo, GateID: f.gate.ID})
	if err != nil {
		t.Fatalf("check out: %v", err)
	}
	if checkedOut.Message != "出园成功" {
		t.Fatalf("check-out message = %q", checkedOut.Message)
	}
	if _, err := f.service.CheckOut(context.Background(), ticket.CheckInput{TicketNo: sold.TicketNo, GateID: f.gate.ID}); !errors.Is(err, ticket.ErrConflict) {
		t.Fatalf("duplicate check-out error = %v, want conflict", err)
	}

	var stored models.Ticket
	if err := f.db.Where("ticket_no = ?", sold.TicketNo).First(&stored).Error; err != nil {
		t.Fatalf("load ticket: %v", err)
	}
	if stored.Status != models.TicketStatusUsed || stored.CheckInTime == nil || stored.CheckOutTime == nil {
		t.Fatalf("unexpected persisted ticket: %+v", stored)
	}
	var recordCount int64
	f.db.Model(&models.CheckRecord{}).Where("ticket_id = ?", stored.ID).Count(&recordCount)
	if recordCount != 2 {
		t.Fatalf("check record count = %d, want 2", recordCount)
	}
}

func TestSellRejectsFullSlotWithoutPartialWrites(t *testing.T) {
	f := newFixture(t)
	full := models.SlotCapacity{Date: f.visitDate, TimeSlotID: 2, MaxCapacity: 1, SoldCount: 1}
	if err := f.db.Create(&full).Error; err != nil {
		t.Fatalf("seed full slot: %v", err)
	}

	_, err := f.service.Sell(context.Background(), f.sellInput("13800000002"))
	if !errors.Is(err, ticket.ErrConflict) {
		t.Fatalf("sell error = %v, want conflict", err)
	}
	assertCount(t, f.db, &models.Ticket{}, 0)
	assertCount(t, f.db, &models.PurchaseQuota{}, 0)
	assertCount(t, f.db, &models.DailyCapacity{}, 0)
}

func TestSellEnforcesPhoneLimit(t *testing.T) {
	f := newFixture(t)
	input := f.sellInput("13800000003")
	for i := 0; i < config.MaxTicketsPerPhone; i++ {
		if _, err := f.service.Sell(context.Background(), input); err != nil {
			t.Fatalf("sell ticket %d: %v", i+1, err)
		}
	}
	if _, err := f.service.Sell(context.Background(), input); !errors.Is(err, ticket.ErrConflict) {
		t.Fatalf("limit error = %v, want conflict", err)
	}
	assertCount(t, f.db, &models.Ticket{}, int64(config.MaxTicketsPerPhone))
	var quota models.PurchaseQuota
	if err := f.db.First(&quota).Error; err != nil {
		t.Fatalf("load quota: %v", err)
	}
	if quota.ActiveCount != config.MaxTicketsPerPhone {
		t.Fatalf("active quota = %d, want %d", quota.ActiveCount, config.MaxTicketsPerPhone)
	}
}

func TestRefundRestoresCapacityAndQuota(t *testing.T) {
	f := newFixture(t)
	sold, err := f.service.Sell(context.Background(), f.sellInput("13800000004"))
	if err != nil {
		t.Fatalf("sell ticket: %v", err)
	}
	if err := f.service.Refund(context.Background(), sold.TicketNo); err != nil {
		t.Fatalf("refund ticket: %v", err)
	}

	var value models.Ticket
	f.db.Where("ticket_no = ?", sold.TicketNo).First(&value)
	if value.Status != models.TicketStatusRefunded {
		t.Fatalf("ticket status = %q", value.Status)
	}
	var daily models.DailyCapacity
	var slot models.SlotCapacity
	var quota models.PurchaseQuota
	f.db.First(&daily)
	f.db.First(&slot)
	f.db.First(&quota)
	if daily.SoldCount != 0 || slot.SoldCount != 0 || quota.ActiveCount != 0 {
		t.Fatalf("counts after refund: daily=%d slot=%d quota=%d", daily.SoldCount, slot.SoldCount, quota.ActiveCount)
	}
}

func TestSellRollsBackWhenLaterWriteFails(t *testing.T) {
	f := newFixture(t)
	trigger := fmt.Sprintf(`CREATE TRIGGER fail_quota_update BEFORE UPDATE ON %s
		BEGIN SELECT RAISE(ABORT, 'forced quota failure'); END;`, f.db.NamingStrategy.TableName("PurchaseQuota"))
	if err := f.db.Exec(trigger).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	_, err := f.service.Sell(context.Background(), f.sellInput("13800000005"))
	if !errors.Is(err, ticket.ErrStorage) {
		t.Fatalf("sell error = %v, want storage error", err)
	}
	assertCount(t, f.db, &models.Ticket{}, 0)
	assertCount(t, f.db, &models.PurchaseQuota{}, 0)
	assertCount(t, f.db, &models.DailyCapacity{}, 0)
	assertCount(t, f.db, &models.SlotCapacity{}, 0)
}

func TestCanceledSellDoesNotPersist(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.service.Sell(ctx, f.sellInput("13800000006"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("sell error = %v, want context canceled", err)
	}
	assertCount(t, f.db, &models.Ticket{}, 0)
	assertCount(t, f.db, &models.PurchaseQuota{}, 0)
	assertCount(t, f.db, &models.DailyCapacity{}, 0)
	assertCount(t, f.db, &models.SlotCapacity{}, 0)
}

func assertCount(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	var got int64
	if err := db.Model(model).Count(&got).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if got != want {
		t.Fatalf("count %T = %d, want %d", model, got, want)
	}
}
