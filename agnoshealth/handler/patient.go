package handler

import (
	"net/http"
	"time"

	"health-backend/external"
	"health-backend/agnoshealth/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PatientHandler handles patient-related HTTP requests.
type PatientHandler struct {
	db  *gorm.DB
	his external.HISSearcher
}

// NewPatientHandler creates a new PatientHandler.
func NewPatientHandler(db *gorm.DB, his external.HISSearcher) *PatientHandler {
	return &PatientHandler{db: db, his: his}
}

// SearchRequest supports both GET query params (form) and POST JSON (json).
type SearchRequest struct {
	NationalID  string `form:"national_id" json:"national_id"`
	PassportID  string `form:"passport_id" json:"passport_id"`
	FirstName   string `form:"first_name" json:"first_name"`
	MiddleName  string `form:"middle_name" json:"middle_name"`
	LastName    string `form:"last_name" json:"last_name"`
	DateOfBirth string `form:"date_of_birth" json:"date_of_birth"`
	PhoneNumber string `form:"phone_number" json:"phone_number"`
	Email       string `form:"email" json:"email"`
}

// Search handles GET and POST /patient/search.
func (h *PatientHandler) Search(c *gin.Context) {

	// Get hospital_id safely from JWT
	hospitalIDVal, exists := c.Get("hospital_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "failed",
			"error":  "unauthorized",
		})
		return
	}

	hospitalID, ok := hospitalIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "failed",
			"error":  "invalid token data",
		})
		return
	}

	// Bind request (GET + POST)
	var req SearchRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// External API fetch (ONLY if ID provided)
	if req.NationalID != "" || req.PassportID != "" {
		searchID := req.NationalID
		if searchID == "" {
			searchID = req.PassportID
		}

		hisPat, err := h.his.SearchPatient(c.Request.Context(), searchID)
		if err == nil && hisPat != nil {

			// Parse DOB safely
			var dob time.Time
			if hisPat.DateOfBirth != "" {
				if parsed, err := time.Parse("2006-01-02", hisPat.DateOfBirth); err == nil {
					dob = parsed
				}
			}

			patient := model.Patient{
				FirstNameTH:  hisPat.FirstNameTH,
				MiddleNameTH: hisPat.MiddleNameTH,
				LastNameTH:   hisPat.LastNameTH,
				FirstNameEN:  hisPat.FirstNameEN,
				MiddleNameEN: hisPat.MiddleNameEN,
				LastNameEN:   hisPat.LastNameEN,
				DateOfBirth:  dob,
				PatientHN:    hisPat.PatientHN,
				NationalID:   hisPat.NationalID,
				PassportID:   hisPat.PassportID,
				PhoneNumber:  hisPat.PhoneNumber,
				Email:        hisPat.Email,
				Gender:       hisPat.Gender,
				HospitalID:   hospitalID,
			}

			// Conflict check (important edge case)
			if patient.NationalID != "" && patient.PassportID != "" {
				var count int64
				h.db.Model(&model.Patient{}).
					Where("hospital_id = ? AND (national_id = ? OR passport_id = ?)",
						hospitalID, patient.NationalID, patient.PassportID).
					Count(&count)

				if count > 1 {
					c.JSON(http.StatusConflict, gin.H{
						"status": "failed",
						"error":  "conflicting patient identifiers",
					})
					return
				}
			}

			// UPSERT logic
			var existing model.Patient
			query := h.db.Where("hospital_id = ?", hospitalID)

			if patient.NationalID != "" {
				query = query.Where("national_id = ?", patient.NationalID)
			} else if patient.PassportID != "" {
				query = query.Where("passport_id = ?", patient.PassportID)
			}

			err := query.First(&existing).Error

			if err == nil {
				// Update existing
				if err := h.db.Model(&existing).Updates(patient).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"status": "failed",
						"error":  "failed to update patient",
					})
					return
				}
			} else if err == gorm.ErrRecordNotFound {
				// Create new
				if err := h.db.Create(&patient).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"status": "failed",
						"error":  "failed to create patient",
					})
					return
				}
			}
		}
	}

	// Build query
	query := h.db.Model(&model.Patient{}).
		Where("hospital_id = ?", hospitalID)

	// Filters
	if req.NationalID != "" {
		query = query.Where("national_id = ?", req.NationalID)
	}
	if req.PassportID != "" {
		query = query.Where("passport_id = ?", req.PassportID)
	}
	if req.FirstName != "" {
		query = query.Where(
			"(first_name_th ILIKE ? OR first_name_en ILIKE ?)",
			"%"+req.FirstName+"%", "%"+req.FirstName+"%",
		)
	}
	if req.MiddleName != "" {
		query = query.Where(
			"(middle_name_th ILIKE ? OR middle_name_en ILIKE ?)",
			"%"+req.MiddleName+"%", "%"+req.MiddleName+"%",
		)
	}
	if req.LastName != "" {
		query = query.Where(
			"(last_name_th ILIKE ? OR last_name_en ILIKE ?)",
			"%"+req.LastName+"%", "%"+req.LastName+"%",
		)
	}
	if req.DateOfBirth != "" {
		if dob, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			query = query.Where("date_of_birth = ?", dob)
		}
	}
	if req.PhoneNumber != "" {
		query = query.Where("phone_number = ?", req.PhoneNumber)
	}
	if req.Email != "" {
		query = query.Where("email = ?", req.Email)
	}

	// Execute with LIMIT
	var patients []model.Patient
	if err := query.Limit(50).Find(&patients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  "failed to search patients",
		})
		return
	}

	// Response
	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"count":    len(patients),
		"patients": patients,
	})
}
