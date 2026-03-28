package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"health-backend/external"
	"health-backend/agnoshealth/handler"
	"health-backend/agnoshealth/middleware"
	"health-backend/agnoshealth/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRouter(db *gorm.DB, jwtSecret string, hisClient external.HISSearcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	staffHandler := handler.NewStaffHandler(db, jwtSecret)
	patientHandler := handler.NewPatientHandler(db, hisClient)

	r.POST("/staff/create", staffHandler.Create)
	r.POST("/staff/login", staffHandler.Login)

	protected := r.Group("/")
	
	protected.Use(middleware.Auth(jwtSecret))
	{
		protected.GET("/patient/search", patientHandler.Search)
		protected.POST("/patient/search", patientHandler.Search)
	}

	return r
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.Hospital{},
		&model.Staff{},
		&model.Patient{},
	))

	return db
}

const testSecret = "testing-default-secret"

// Helper to create hospital
func createHospital(t *testing.T, db *gorm.DB, name string) model.Hospital {
	h := model.Hospital{Name: name}
	require.NoError(t, db.Create(&h).Error)
	return h
}

func TestCreateStaff_Success(t *testing.T) {
	db := newTestDB(t)

	// must create hospital first
	createHospital(t, db, "Mission Hospital")

	r := setupRouter(db, testSecret, nil)

	body := `{"username":"Chanya","password":"Chanya123","hospital":"Mission Hospital"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, "Chanya", resp["username"])
	assert.Equal(t, "Mission Hospital", resp["hospital"])
}

func TestCreateStaff_Duplicate(t *testing.T) {
	db := newTestDB(t)
	createHospital(t, db, "Mission Hospital")

	r := setupRouter(db, testSecret, nil)

	body := `{"username":"Anurak","password":"Anurak123","hospital":"Mission Hospital"}`

	req1 := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestCreateStaff_MissingFields(t *testing.T) {
	db := newTestDB(t)
	r := setupRouter(db, testSecret, nil)

	body := `{"username":"Pimchanok"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Success(t *testing.T) {
	db := newTestDB(t)
	createHospital(t, db, "Bangkok Hospital")

	r := setupRouter(db, testSecret, nil)

	createBody := `{"username":"Rinrada","password":"Rinrada123","hospital":"Bangkok Hospital"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	loginBody := `{"username":"Rinrada","password":"Rinrada123","hospital":"Bangkok Hospital"}`
	req2 := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(loginBody))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp["token"])
}

func TestLogin_WrongPassword(t *testing.T) {
	db := newTestDB(t)
	createHospital(t, db, "Siam Care Hospital")

	r := setupRouter(db, testSecret, nil)

	createBody := `{"username":"Rinrada","password":"correct123","hospital":"Siam Care Hospital"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	loginBody := `{"username":"Rinrada","password":"wrong123","hospital":"Siam Care Hospital"}`
	req2 := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(loginBody))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestLogin_WrongHospital(t *testing.T) {
	db := newTestDB(t)

	createHospital(t, db, "Mission Hospital")
	createHospital(t, db, "Bangkok Hospital")

	r := setupRouter(db, testSecret, nil)

	createBody := `{"username":"Thanawat","password":"Thanawat","hospital":"Mission Hospital"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	loginBody := `{"username":"Thanawat","password":"Thanawat","hospital":"Bangkok Hospital"}`
	req2 := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(loginBody))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}
