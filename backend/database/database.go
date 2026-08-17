package database

import (
	"fmt"
	"log"
	"scenic-ticket/config"
	"scenic-ticket/models"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

const slotCapacityUniqueIndex = "ux_slot_capacities_date_time_slot"

func Init(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := Migrate(db); err != nil {
		return err
	}
	if err := seedData(db); err != nil {
		return err
	}

	DB = db
	log.Println("Database initialized successfully")
	return nil
}

func Migrate(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable(&models.SlotCapacity{}) {
			if err := rejectDuplicateSlotCapacities(tx); err != nil {
				return err
			}
		}
		if err := tx.AutoMigrate(
			&models.User{},
			&models.TicketType{},
			&models.Gate{},
			&models.Ticket{},
			&models.CheckRecord{},
			&models.DailyCapacity{},
			&models.SlotCapacity{},
			&models.PurchaseQuota{},
		); err != nil {
			return fmt.Errorf("auto-migrate schema: %w", err)
		}
		return ensureSlotCapacityUniqueIndex(tx)
	})
}

func rejectDuplicateSlotCapacities(db *gorm.DB) error {
	var duplicate struct {
		Date       time.Time
		TimeSlotID int
		RowCount   int64
	}
	result := db.Raw(`
		SELECT date, time_slot_id, COUNT(*) AS row_count
		FROM slot_capacities
		GROUP BY date, time_slot_id
		HAVING COUNT(*) > 1
		ORDER BY row_count DESC, date, time_slot_id
		LIMIT 1
	`).Scan(&duplicate)
	if result.Error != nil {
		return fmt.Errorf("inspect slot capacity duplicates: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		return fmt.Errorf(
			"slot capacity migration blocked: duplicate rows for date=%s time_slot_id=%d rows=%d",
			config.DateKey(duplicate.Date).Format(config.DateLayout),
			duplicate.TimeSlotID,
			duplicate.RowCount,
		)
	}
	return nil
}

func ensureSlotCapacityUniqueIndex(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS ux_slot_capacities_date_time_slot
		ON slot_capacities (date, time_slot_id)
	`).Error; err != nil {
		return fmt.Errorf("create slot capacity unique index: %w", err)
	}

	var valid bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_class table_class
			JOIN pg_namespace namespace ON namespace.oid = table_class.relnamespace
			JOIN pg_index index_meta ON index_meta.indrelid = table_class.oid
			JOIN pg_class index_class ON index_class.oid = index_meta.indexrelid
			WHERE namespace.nspname = current_schema()
			  AND table_class.relname = 'slot_capacities'
			  AND index_class.relname = ?
			  AND index_meta.indisunique
			  AND index_meta.indisvalid
			  AND index_meta.indpred IS NULL
			  AND index_meta.indnkeyatts = 2
			  AND pg_get_indexdef(index_meta.indexrelid, 1, true) = 'date'
			  AND pg_get_indexdef(index_meta.indexrelid, 2, true) = 'time_slot_id'
		)
	`, slotCapacityUniqueIndex).Scan(&valid).Error; err != nil {
		return fmt.Errorf("verify slot capacity unique index: %w", err)
	}
	if !valid {
		return fmt.Errorf("slot capacity unique index %q is missing or invalid", slotCapacityUniqueIndex)
	}
	return nil
}

func seedData(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedUsers(tx); err != nil {
			return err
		}
		if err := seedTicketTypes(tx); err != nil {
			return err
		}
		return seedGates(tx)
	})
}

func seedUsers(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	adminPwd, err := bcrypt.GenerateFromPassword([]byte("scenic2024"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	sellerPwd, err := bcrypt.GenerateFromPassword([]byte("sell123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seller password: %w", err)
	}

	users := []models.User{
		{Username: "admin", Password: string(adminPwd), Name: "系统管理员", Role: models.RoleAdmin},
		{Username: "seller1", Password: string(sellerPwd), Name: "售票员1号", Role: models.RoleSeller},
		{Username: "seller2", Password: string(sellerPwd), Name: "售票员2号", Role: models.RoleSeller},
	}

	if err := db.Create(&users).Error; err != nil {
		return fmt.Errorf("seed users: %w", err)
	}
	log.Println("Users seeded")
	return nil
}

func seedTicketTypes(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.TicketType{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count ticket types: %w", err)
	}
	if count > 0 {
		return nil
	}

	ticketTypes := []models.TicketType{
		{Name: "成人票", Price: 120.0, Description: "成人全价票", Status: 1},
		{Name: "学生票", Price: 60.0, Description: "学生半价票，需持学生证", Status: 1},
		{Name: "老人免费票", Price: 0.0, Description: "65岁以上老人免费，需持身份证", Status: 1},
	}

	if err := db.Create(&ticketTypes).Error; err != nil {
		return fmt.Errorf("seed ticket types: %w", err)
	}
	log.Println("Ticket types seeded")
	return nil
}

func seedGates(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Gate{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count gates: %w", err)
	}
	if count > 0 {
		return nil
	}

	gates := []models.Gate{
		{Name: "东门1号", Location: "景区东门", Status: 1},
		{Name: "东门2号", Location: "景区东门", Status: 1},
		{Name: "西门1号", Location: "景区西门", Status: 1},
		{Name: "南门", Location: "景区南门", Status: 1},
		{Name: "北门", Location: "景区北门", Status: 1},
	}

	if err := db.Create(&gates).Error; err != nil {
		return fmt.Errorf("seed gates: %w", err)
	}
	log.Println("Gates seeded")
	return nil
}
