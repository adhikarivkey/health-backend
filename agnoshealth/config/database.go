package config

import (
	"fmt"

	"health-backend/agnoshealth/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Bangkok",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	if err := db.AutoMigrate(
		&model.Hospital{},
		&model.Staff{},
		&model.Patient{},
	); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}
