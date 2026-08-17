package database

import (
	"context"
	"errors"
	"scenic-ticket/config"
	"scenic-ticket/models"
	"scenic-ticket/ticket"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

func (r *TicketRepository) WithTx(ctx context.Context, fn func(ticket.TxRepository) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&ticketTx{db: tx})
	})
}

type ticketTx struct {
	db *gorm.DB
}

func (tx *ticketTx) FindTicketType(ctx context.Context, id uint) (*models.TicketType, error) {
	var value models.TicketType
	err := tx.db.WithContext(ctx).Where("id = ? AND status = ?", id, 1).First(&value).Error
	return &value, translateNotFound(err)
}

func (tx *ticketTx) GetOrCreatePurchaseQuota(ctx context.Context, date time.Time, phone string) (*models.PurchaseQuota, error) {
	seed := models.PurchaseQuota{Date: date, Phone: phone}
	created := tx.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "date"}, {Name: "phone"}},
		DoNothing: true,
	}).Create(&seed)
	if created.Error != nil {
		return nil, created.Error
	}

	var value models.PurchaseQuota
	if err := tx.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("date = ? AND phone = ?", date, phone).First(&value).Error; err != nil {
		return nil, err
	}
	if created.RowsAffected == 1 {
		var count int64
		if err := tx.db.WithContext(ctx).Model(&models.Ticket{}).
			Where("phone = ? AND visit_date = ? AND status <> ?", phone, date, models.TicketStatusRefunded).
			Count(&count).Error; err != nil {
			return nil, err
		}
		value.ActiveCount = int(count)
		if count > 0 {
			if err := tx.db.WithContext(ctx).Save(&value).Error; err != nil {
				return nil, err
			}
		}
	}
	return &value, nil
}

func (tx *ticketTx) GetOrCreateDailyCapacity(ctx context.Context, date time.Time) (*models.DailyCapacity, error) {
	seed := models.DailyCapacity{Date: date, MaxCapacity: config.MaxDailyCapacity}
	if err := tx.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "date"}},
		DoNothing: true,
	}).Create(&seed).Error; err != nil {
		return nil, err
	}

	var value models.DailyCapacity
	err := tx.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("date = ?", date).First(&value).Error
	return &value, err
}

func (tx *ticketTx) GetOrCreateSlotCapacity(ctx context.Context, date time.Time, slotID int) (*models.SlotCapacity, error) {
	seed := models.SlotCapacity{Date: date, TimeSlotID: slotID, MaxCapacity: config.DefaultSlotCapacity}
	if err := tx.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "date"}, {Name: "time_slot_id"}},
		DoNothing: true,
	}).Create(&seed).Error; err != nil {
		return nil, err
	}

	var value models.SlotCapacity
	err := tx.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("date = ? AND time_slot_id = ?", date, slotID).First(&value).Error
	return &value, err
}

func (tx *ticketTx) FindTicketForUpdate(ctx context.Context, ticketNo string) (*models.Ticket, error) {
	var value models.Ticket
	err := tx.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("ticket_no = ?", ticketNo).First(&value).Error
	return &value, translateNotFound(err)
}

func (tx *ticketTx) FindGate(ctx context.Context, id uint) (*models.Gate, error) {
	var value models.Gate
	err := tx.db.WithContext(ctx).Where("id = ? AND status = ?", id, 1).First(&value).Error
	return &value, translateNotFound(err)
}

func (tx *ticketTx) CountCheckedOut(ctx context.Context, date time.Time) (int64, error) {
	var count int64
	start, end := config.DayRange(date)
	err := tx.db.WithContext(ctx).Model(&models.CheckRecord{}).
		Where("check_type = ? AND check_time >= ? AND check_time < ?", models.CheckTypeOut, start, end).
		Count(&count).Error
	return count, err
}

func (tx *ticketTx) CreateTicket(ctx context.Context, value *models.Ticket) error {
	return tx.db.WithContext(ctx).Omit("TicketType", "CheckInGate", "CheckOutGate").Create(value).Error
}

func (tx *ticketTx) SaveTicket(ctx context.Context, value *models.Ticket) error {
	return tx.db.WithContext(ctx).Omit("TicketType", "CheckInGate", "CheckOutGate").Save(value).Error
}

func (tx *ticketTx) SavePurchaseQuota(ctx context.Context, value *models.PurchaseQuota) error {
	return tx.db.WithContext(ctx).Save(value).Error
}

func (tx *ticketTx) SaveDailyCapacity(ctx context.Context, value *models.DailyCapacity) error {
	return tx.db.WithContext(ctx).Save(value).Error
}

func (tx *ticketTx) SaveSlotCapacity(ctx context.Context, value *models.SlotCapacity) error {
	return tx.db.WithContext(ctx).Save(value).Error
}

func (tx *ticketTx) CreateCheckRecord(ctx context.Context, value *models.CheckRecord) error {
	return tx.db.WithContext(ctx).Create(value).Error
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ticket.ErrRecordNotFound
	}
	return err
}
