package config

import (
	"os"
)

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
