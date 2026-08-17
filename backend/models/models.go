package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	Name      string         `gorm:"size:50" json:"name"`
	Role      string         `gorm:"size:20;not null" json:"role"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

const (
	RoleAdmin  = "admin"
	RoleSeller = "seller"
)

type TicketType struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:50;not null" json:"name"`
	Price       float64        `gorm:"not null;default:0" json:"price"`
	Description string         `gorm:"size:255" json:"description"`
	Status      int            `gorm:"default:1" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Gate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Location  string    `gorm:"size:100" json:"location"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Ticket struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	TicketNo       string         `gorm:"uniqueIndex;size:20;not null" json:"ticket_no"`
	TicketTypeID   uint           `gorm:"not null" json:"ticket_type_id"`
	TicketType     TicketType     `gorm:"foreignKey:TicketTypeID" json:"ticket_type"`
	BuyerName      string         `gorm:"size:50;not null" json:"buyer_name"`
	Phone          string         `gorm:"size:20;not null;index" json:"phone"`
	VisitDate      time.Time      `gorm:"not null;index" json:"visit_date"`
	TimeSlotID     int            `gorm:"not null;default:1" json:"time_slot_id"`
	TimeSlotName   string         `gorm:"size:20" json:"time_slot_name"`
	Price          float64        `gorm:"not null" json:"price"`
	Status         string         `gorm:"size:20;not null;default:unused" json:"status"`
	SellerID       uint           `gorm:"not null" json:"seller_id"`
	SellerName     string         `gorm:"size:50" json:"seller_name"`
	CheckInGateID  *uint          `json:"check_in_gate_id"`
	CheckInGate    *Gate          `gorm:"foreignKey:CheckInGateID" json:"check_in_gate"`
	CheckInTime    *time.Time     `json:"check_in_time"`
	CheckOutGateID *uint          `json:"check_out_gate_id"`
	CheckOutGate   *Gate          `gorm:"foreignKey:CheckOutGateID" json:"check_out_gate"`
	CheckOutTime   *time.Time     `json:"check_out_time"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

const (
	TicketStatusUnused   = "unused"
	TicketStatusUsed     = "used"
	TicketStatusRefunded = "refunded"
)

type CheckRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TicketID   uint      `gorm:"not null;index;uniqueIndex:idx_ticket_check_type" json:"ticket_id"`
	TicketNo   string    `gorm:"size:20;index" json:"ticket_no"`
	GateID     uint      `gorm:"not null" json:"gate_id"`
	GateName   string    `gorm:"size:50" json:"gate_name"`
	CheckType  string    `gorm:"size:10;not null;uniqueIndex:idx_ticket_check_type" json:"check_type"`
	CheckTime  time.Time `gorm:"not null;index" json:"check_time"`
	TimeSlotID int       `json:"time_slot_id"`
	CreatedAt  time.Time `json:"created_at"`
}

const (
	CheckTypeIn  = "in"
	CheckTypeOut = "out"
)

type DailyCapacity struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Date           time.Time `gorm:"uniqueIndex;not null" json:"date"`
	MaxCapacity    int       `gorm:"not null;default:8000" json:"max_capacity"`
	SoldCount      int       `gorm:"default:0" json:"sold_count"`
	CheckedInCount int       `gorm:"default:0" json:"checked_in_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SlotCapacity struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Date           time.Time `gorm:"not null;uniqueIndex:ux_slot_capacities_date_time_slot" json:"date"`
	TimeSlotID     int       `gorm:"not null;uniqueIndex:ux_slot_capacities_date_time_slot" json:"time_slot_id"`
	MaxCapacity    int       `gorm:"not null;default:2000" json:"max_capacity"`
	SoldCount      int       `gorm:"default:0" json:"sold_count"`
	CheckedInCount int       `gorm:"default:0" json:"checked_in_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PurchaseQuota struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Date        time.Time `gorm:"not null;uniqueIndex:idx_quota_date_phone" json:"date"`
	Phone       string    `gorm:"size:20;not null;uniqueIndex:idx_quota_date_phone" json:"phone"`
	ActiveCount int       `gorm:"not null;default:0" json:"active_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
