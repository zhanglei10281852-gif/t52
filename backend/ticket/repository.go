package ticket

import (
	"context"
	"scenic-ticket/models"
	"time"
)

type Repository interface {
	WithTx(ctx context.Context, fn func(TxRepository) error) error
}

type TxRepository interface {
	FindTicketType(ctx context.Context, id uint) (*models.TicketType, error)
	GetOrCreatePurchaseQuota(ctx context.Context, date time.Time, phone string) (*models.PurchaseQuota, error)
	GetOrCreateDailyCapacity(ctx context.Context, date time.Time) (*models.DailyCapacity, error)
	GetOrCreateSlotCapacity(ctx context.Context, date time.Time, slotID int) (*models.SlotCapacity, error)
	FindTicketForUpdate(ctx context.Context, ticketNo string) (*models.Ticket, error)
	FindGate(ctx context.Context, id uint) (*models.Gate, error)
	CountCheckedOut(ctx context.Context, date time.Time) (int64, error)
	CreateTicket(ctx context.Context, value *models.Ticket) error
	SaveTicket(ctx context.Context, value *models.Ticket) error
	SavePurchaseQuota(ctx context.Context, value *models.PurchaseQuota) error
	SaveDailyCapacity(ctx context.Context, value *models.DailyCapacity) error
	SaveSlotCapacity(ctx context.Context, value *models.SlotCapacity) error
	CreateCheckRecord(ctx context.Context, value *models.CheckRecord) error
}
