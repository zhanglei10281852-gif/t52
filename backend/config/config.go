package config

import (
	"os"
	"time"
)

const DateLayout = "2006-01-02"

var BusinessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	Port       string
}

func Load() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "scenic"),
		DBPassword: getEnv("DB_PASSWORD", "scenic2024"),
		DBName:     getEnv("DB_NAME", "scenic_db"),
		JWTSecret:  getEnv("JWT_SECRET", "scenic-ticket-jwt-secret-2024"),
		Port:       getEnv("PORT", "8741"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// DateKey stores a business calendar date as UTC midnight for compatibility
// with existing date-only database columns.
func DateKey(value time.Time) time.Time {
	local := value.In(BusinessLocation)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func ParseDate(value string) (time.Time, error) {
	parsed, err := time.Parse(DateLayout, value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), nil
}

// DayRange returns the actual instants covered by a business calendar date.
func DayRange(value time.Time) (time.Time, time.Time) {
	key := value.In(time.UTC)
	start := time.Date(key.Year(), key.Month(), key.Day(), 0, 0, 0, 0, BusinessLocation)
	return start, start.AddDate(0, 0, 1)
}

const (
	MaxDailyCapacity    = 8000
	CapacityOverSell    = 1.1
	MaxTicketsPerPhone  = 5
	DefaultSlotCapacity = 2000
)

var TimeSlots = []TimeSlotInfo{
	{ID: 1, Name: "08:00-10:00", StartHour: 8, EndHour: 10},
	{ID: 2, Name: "10:00-12:00", StartHour: 10, EndHour: 12},
	{ID: 3, Name: "12:00-14:00", StartHour: 12, EndHour: 14},
	{ID: 4, Name: "14:00-16:00", StartHour: 14, EndHour: 16},
}

type TimeSlotInfo struct {
	ID        int
	Name      string
	StartHour int
	EndHour   int
}

func GetTimeSlotInfo(slotID int) *TimeSlotInfo {
	for _, slot := range TimeSlots {
		if slot.ID == slotID {
			return &slot
		}
	}
	return nil
}
