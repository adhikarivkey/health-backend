package model

import (
	"time"

	"gorm.io/gorm"
)

type Patient struct {
	ID uint `gorm:"primaryKey" json:"id"`

	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`

	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`

	DateOfBirth time.Time `gorm:"type:date" json:"date_of_birth"`

	PatientHN string `gorm:"index" json:"patient_hn"`

	NationalID string `gorm:"uniqueIndex:idx_national_hospital" json:"national_id"`
	PassportID string `gorm:"uniqueIndex:idx_passport_hospital" json:"passport_id"`

	PhoneNumber string `gorm:"json:"phone_number"`
	Email       string `gorm:"json:"email"`
	Gender      string `gorm:"type:char(1)" json:"gender"`

	// Foreign key reference to Hospital
	HospitalID uint     `gorm:"not null;uniqueIndex:idx_national_hospital;uniqueIndex:idx_passport_hospital"`
	Hospital   Hospital `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"hospital"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
