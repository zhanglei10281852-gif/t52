//go:build integration

package database_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"scenic-ticket/config"
	"scenic-ticket/database"
	"scenic-ticket/models"
	"scenic-ticket/ticket"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var schemaSequence atomic.Uint64

type integrationClock struct {
	now time.Time
}

func (c integrationClock) Now() time.Time {
	return c.now
}

type integrationNumbers struct {
	next atomic.Uint64
}

func (g *integrationNumbers) Generate(date time.Time) (string, error) {
	return fmt.Sprintf("%s%012d", date.Format("20060102"), g.next.Add(1)), nil
}

func TestMigrationUpgradesLegacySlotIndexAndSellWorks(t *testing.T) {
	db := newPostgresSchema(t)
	createLegacySlotCapacityTable(t, db)

	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	if unique := indexIsUnique(t, db, "idx_date_slot"); unique {
		t.Fatal("legacy idx_date_slot unexpectedly changed to unique")
	}
	if unique := indexIsUnique(t, db, "ux_slot_capacities_date_time_slot"); !unique {
		t.Fatal("new slot capacity index is not unique")
	}

	ticketType := seedTicketType(t, db)
	service := newIntegrationService(db)
	visitDate := mustDate(t, "2026-08-17")
	result, err := service.Sell(context.Background(), sellInput(ticketType.ID, visitDate, "13800010001"))
	if err != nil {
		t.Fatalf("sell through migrated repository: %v", err)
	}
	if result.TicketNo == "" {
		t.Fatal("sell returned an empty ticket number")
	}
}

func TestMigrationRejectsDuplicateLegacyRowsWithoutChangingData(t *testing.T) {
	db := newPostgresSchema(t)
	createLegacySlotCapacityTable(t, db)
	if err := db.Exec(`
		INSERT INTO slot_capacities (date, time_slot_id, max_capacity, sold_count, checked_in_count)
		VALUES
			('2026-08-17 00:00:00+00', 2, 2000, 1, 0),
			('2026-08-17 00:00:00+00', 2, 2000, 1, 0)
	`).Error; err != nil {
		t.Fatalf("seed duplicate rows: %v", err)
	}

	err := database.Migrate(db)
	if err == nil {
		t.Fatal("migration succeeded with duplicate slot capacity rows")
	}
	message := err.Error()
	for _, want := range []string{"date=2026-08-17", "time_slot_id=2", "rows=2"} {
		if !strings.Contains(message, want) {
			t.Fatalf("migration error %q does not contain %q", message, want)
		}
	}

	var count int64
	if err := db.Table("slot_capacities").Count(&count).Error; err != nil {
		t.Fatalf("count preserved rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("slot capacity rows after failed migration = %d, want 2", count)
	}
}

func TestConcurrentSalesEnforcePhoneLimit(t *testing.T) {
	db := newPostgresSchema(t)
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	ticketType := seedTicketType(t, db)
	service := newIntegrationService(db)
	visitDate := mustDate(t, "2026-08-17")

	errs := runConcurrentSales(12, func(index int) ticket.SellInput {
		return sellInput(ticketType.ID, visitDate, "13800010002")
	}, service)
	assertSaleOutcomes(t, errs, config.MaxTicketsPerPhone)

	var quota models.PurchaseQuota
	if err := db.Where("date = ? AND phone = ?", visitDate, "13800010002").First(&quota).Error; err != nil {
		t.Fatalf("load purchase quota: %v", err)
	}
	if quota.ActiveCount != config.MaxTicketsPerPhone {
		t.Fatalf("active quota = %d, want %d", quota.ActiveCount, config.MaxTicketsPerPhone)
	}
}

func TestConcurrentSalesDoNotExceedSlotCapacity(t *testing.T) {
	db := newPostgresSchema(t)
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	ticketType := seedTicketType(t, db)
	service := newIntegrationService(db)
	visitDate := mustDate(t, "2026-08-17")
	const capacity = 3
	if err := db.Create(&models.SlotCapacity{
		Date: visitDate, TimeSlotID: 2, MaxCapacity: capacity,
	}).Error; err != nil {
		t.Fatalf("seed slot capacity: %v", err)
	}

	errs := runConcurrentSales(12, func(index int) ticket.SellInput {
		return sellInput(ticketType.ID, visitDate, fmt.Sprintf("1380002%04d", index))
	}, service)
	assertSaleOutcomes(t, errs, capacity)

	var stored models.SlotCapacity
	if err := db.Where("date = ? AND time_slot_id = ?", visitDate, 2).First(&stored).Error; err != nil {
		t.Fatalf("load slot capacity: %v", err)
	}
	if stored.SoldCount != capacity {
		t.Fatalf("slot sold count = %d, want %d", stored.SoldCount, capacity)
	}
	var tickets int64
	if err := db.Model(&models.Ticket{}).Where("visit_date = ? AND time_slot_id = ?", visitDate, 2).Count(&tickets).Error; err != nil {
		t.Fatalf("count sold tickets: %v", err)
	}
	if tickets != capacity {
		t.Fatalf("sold tickets = %d, want %d", tickets, capacity)
	}
}

func newPostgresSchema(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=scenic password=scenic2024 dbname=scenic_db sslmode=disable"
	}
	quiet := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	admin, err := gorm.Open(postgres.Open(dsn), quiet)
	if err != nil {
		t.Fatalf("open postgres admin connection: %v", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("open postgres admin pool: %v", err)
	}

	schema := fmt.Sprintf("ticket_it_%d_%d", os.Getpid(), schemaSequence.Add(1))
	if err := admin.Exec("CREATE SCHEMA " + quoteIdentifier(schema)).Error; err != nil {
		_ = adminSQL.Close()
		t.Fatalf("create test schema: %v", err)
	}

	var schemaSQL *sql.DB
	t.Cleanup(func() {
		if schemaSQL != nil {
			_ = schemaSQL.Close()
		}
		if err := admin.Exec("DROP SCHEMA IF EXISTS " + quoteIdentifier(schema) + " CASCADE").Error; err != nil {
			t.Errorf("drop test schema: %v", err)
		}
		_ = adminSQL.Close()
	})

	db, err := gorm.Open(postgres.Open(dsnWithSearchPath(dsn, schema)), quiet)
	if err != nil {
		t.Fatalf("open schema connection: %v", err)
	}
	schemaPool, err := db.DB()
	if err != nil {
		t.Fatalf("open schema pool: %v", err)
	}
	schemaPool.SetMaxOpenConns(20)
	schemaSQL = schemaPool
	return db
}

func dsnWithSearchPath(dsn, schema string) string {
	if strings.Contains(dsn, "://") {
		parsed, err := url.Parse(dsn)
		if err == nil {
			query := parsed.Query()
			query.Set("search_path", schema)
			parsed.RawQuery = query.Encode()
			return parsed.String()
		}
	}
	return dsn + " search_path=" + schema
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func createLegacySlotCapacityTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
		CREATE TABLE slot_capacities (
			id bigserial PRIMARY KEY,
			date timestamptz NOT NULL,
			time_slot_id bigint NOT NULL,
			max_capacity bigint NOT NULL DEFAULT 2000,
			sold_count bigint DEFAULT 0,
			checked_in_count bigint DEFAULT 0,
			created_at timestamptz,
			updated_at timestamptz
		)
	`).Error; err != nil {
		t.Fatalf("create legacy slot capacity table: %v", err)
	}
	if err := db.Exec(`CREATE INDEX idx_date_slot ON slot_capacities (date, time_slot_id)`).Error; err != nil {
		t.Fatalf("create legacy slot index: %v", err)
	}
}

func indexIsUnique(t *testing.T, db *gorm.DB, name string) bool {
	t.Helper()
	var unique bool
	result := db.Raw(`
		SELECT index_meta.indisunique
		FROM pg_class index_class
		JOIN pg_namespace namespace ON namespace.oid = index_class.relnamespace
		JOIN pg_index index_meta ON index_meta.indexrelid = index_class.oid
		WHERE namespace.nspname = current_schema() AND index_class.relname = ?
	`, name).Scan(&unique)
	if result.Error != nil {
		t.Fatalf("inspect index %s: %v", name, result.Error)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("index %s was not preserved or created", name)
	}
	return unique
}

func seedTicketType(t *testing.T, db *gorm.DB) models.TicketType {
	t.Helper()
	value := models.TicketType{Name: "成人票", Price: 120, Status: 1}
	if err := db.Create(&value).Error; err != nil {
		t.Fatalf("seed ticket type: %v", err)
	}
	return value
}

func newIntegrationService(db *gorm.DB) *ticket.Service {
	return ticket.NewService(
		database.NewTicketRepository(db),
		integrationClock{now: time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)},
		&integrationNumbers{},
	)
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := config.ParseDate(value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}

func sellInput(ticketTypeID uint, visitDate time.Time, phone string) ticket.SellInput {
	return ticket.SellInput{
		TicketTypeID: ticketTypeID,
		BuyerName:    "集成测试游客",
		Phone:        phone,
		VisitDate:    visitDate,
		TimeSlotID:   2,
		SellerID:     7,
		SellerName:   "售票员",
	}
}

func runConcurrentSales(count int, input func(int) ticket.SellInput, service *ticket.Service) []error {
	start := make(chan struct{})
	errs := make([]error, count)
	var workers sync.WaitGroup
	workers.Add(count)
	for index := 0; index < count; index++ {
		go func(index int) {
			defer workers.Done()
			<-start
			_, errs[index] = service.Sell(context.Background(), input(index))
		}(index)
	}
	close(start)
	workers.Wait()
	return errs
}

func assertSaleOutcomes(t *testing.T, errs []error, wantSuccess int) {
	t.Helper()
	successes := 0
	conflicts := 0
	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ticket.ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected sale error: %v", err)
		}
	}
	if successes != wantSuccess {
		t.Fatalf("successful sales = %d, want %d", successes, wantSuccess)
	}
	if conflicts != len(errs)-wantSuccess {
		t.Fatalf("conflicting sales = %d, want %d", conflicts, len(errs)-wantSuccess)
	}
}
