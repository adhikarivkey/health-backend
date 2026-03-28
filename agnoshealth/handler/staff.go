package handler

import (
	"net/http"
	"time"

	"health-backend/agnoshealth/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	mw "health-backend/agnoshealth/middleware"
)

// StaffHandler handles staff-related HTTP requests.
type StaffHandler struct {
	db        *gorm.DB
	jwtSecret string
}

// NewStaffHandler creates a new StaffHandler.
func NewStaffHandler(db *gorm.DB, jwtSecret string) *StaffHandler {
	return &StaffHandler{db: db, jwtSecret: jwtSecret}
}

// CreateRequest is the request body for staff creation.
type CreateRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"`
}

// LoginRequest is the request body for staff login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"`
}

// Create handles POST /staff/create.
func (h *StaffHandler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Find hospital by name
	var hospital model.Hospital
	if err := h.db.Where("name = ?", req.Hospital).First(&hospital).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  "hospital not found",
		})
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  "failed to hash password",
		})
		return
	}

	// Create staff
	staff := model.Staff{
		Username:   req.Username,
		Password:   string(hashed),
		HospitalID: hospital.ID,
	}

	if err := h.db.Create(&staff).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"status": "failed",
			"error":  "staff username already exists",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          staff.ID,
		"status":      "success",
		"username":    staff.Username,
		"hospital_id": hospital.ID,
		"hospital":    hospital.Name,
	})
}

// Login handles POST /staff/login.
func (h *StaffHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Find hospital
	var hospital model.Hospital
	if err := h.db.Where("name = ?", req.Hospital).First(&hospital).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "failed",
			"error":  "invalid credentials",
		})
		return
	}

	// Find staff by username + hospital_id
	var staff model.Staff
	if err := h.db.
		Where("username = ? AND hospital_id = ?", req.Username, hospital.ID).
		First(&staff).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "failed",
			"error":  "invalid credentials",
		})
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(staff.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "failed",
			"error":  "invalid credentials",
		})
		return
	}

	// Create JWT token
	claims := &mw.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		StaffID:   staff.ID,
		HospitalID: hospital.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"token": signed,
		"hospital": hospital.Name,
	})
}
