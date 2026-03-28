package model

import (
	"time"

	"gorm.io/gorm"
)

type Staff struct {
	ID uint `gorm:"primaryKey" json:"id"`

	Username string `gorm:"not null;uniqueIndex:idx_username_hospital" json:"username"`
	Password string `gorm:"not null" json:"-"`

	// Foreign key reference
	HospitalID uint     `gorm:"not null;uniqueIndex:idx_username_hospital" json:"hospital_id"`
	Hospital   Hospital `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"hospital"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
