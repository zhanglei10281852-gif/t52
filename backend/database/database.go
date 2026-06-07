package database

import (
	"fmt"
	"log"
	"scenic-ticket/config"
	"scenic-ticket/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	err = DB.AutoMigrate(
		&models.User{},
		&models.TicketType{},
		&models.Gate{},
		&models.Ticket{},
		&models.CheckRecord{},
		&models.DailyCapacity{},
		&models.SlotCapacity{},
	)
	if err != nil {
		return err
	}

	seedData()
	log.Println("Database initialized successfully")
	return nil
}

func seedData() {
	seedUsers()
	seedTicketTypes()
	seedGates()
}

func seedUsers() {
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	adminPwd, _ := bcrypt.GenerateFromPassword([]byte("scenic2024"), bcrypt.DefaultCost)
	sellerPwd, _ := bcrypt.GenerateFromPassword([]byte("sell123"), bcrypt.DefaultCost)

	users := []models.User{
		{Username: "admin", Password: string(adminPwd), Name: "系统管理员", Role: models.RoleAdmin},
		{Username: "seller1", Password: string(sellerPwd), Name: "售票员1号", Role: models.RoleSeller},
		{Username: "seller2", Password: string(sellerPwd), Name: "售票员2号", Role: models.RoleSeller},
	}

	for _, u := range users {
		DB.Create(&u)
	}
	log.Println("Users seeded")
}

func seedTicketTypes() {
	var count int64
	DB.Model(&models.TicketType{}).Count(&count)
	if count > 0 {
		return
	}

	ticketTypes := []models.TicketType{
		{Name: "成人票", Price: 120.0, Description: "成人全价票", Status: 1},
		{Name: "学生票", Price: 60.0, Description: "学生半价票，需持学生证", Status: 1},
		{Name: "老人免费票", Price: 0.0, Description: "65岁以上老人免费，需持身份证", Status: 1},
	}

	for _, t := range ticketTypes {
		DB.Create(&t)
	}
	log.Println("Ticket types seeded")
}

func seedGates() {
	var count int64
	DB.Model(&models.Gate{}).Count(&count)
	if count > 0 {
		return
	}

	gates := []models.Gate{
		{Name: "东门1号", Location: "景区东门", Status: 1},
		{Name: "东门2号", Location: "景区东门", Status: 1},
		{Name: "西门1号", Location: "景区西门", Status: 1},
		{Name: "南门", Location: "景区南门", Status: 1},
		{Name: "北门", Location: "景区北门", Status: 1},
	}

	for _, g := range gates {
		DB.Create(&g)
	}
	log.Println("Gates seeded")
}
