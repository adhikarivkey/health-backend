package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"health-backend/external"
	"health-backend/agnoshealth/middleware"
	"health-backend/agnoshealth/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock HIS
type mockHIS struct {
	response *external.PatientResponse
	err      error
	called   bool
	calledID string
}

func (m *mockHIS) SearchPatient(ctx context.Context, id string) (*external.PatientResponse, error) {
	m.called = true
	m.calledID = id
	return m.response, m.err
}

// Generate JWT with hospital_id
func generateToken(t *testing.T, secret string, staffID uint, hospitalID uint) string {
	t.Helper()

	claims := &middleware.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		StaffID:   staffID,
		HospitalID: hospitalID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	return signed
}

func TestSearchPatient_NoAuth(t *testing.T) {
	db := newTestDB(t)
	r := setupRouter(db, testSecret, &mockHIS{})

	req := httptest.NewRequest(http.MethodGet, "/patient/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSearchPatient_Success(t *testing.T) {
	db := newTestDB(t)
	mock := &mockHIS{}
	r := setupRouter(db, testSecret, mock)

	// Create hospital
	hospital := model.Hospital{Name: "Mission Hospital"}
	require.NoError(t, db.Create(&hospital).Error)

	// Seed patient
	patient := model.Patient{
		FirstNameEN: "Vivek",
		LastNameEN:  "Adhikari",
		NationalID:  "1234567890123",
		HospitalID:  hospital.ID,
	}
	require.NoError(t, db.Create(&patient).Error)

	token := generateToken(t, testSecret, 1, hospital.ID)

	req := httptest.NewRequest(http.MethodGet, "/patient/search", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, float64(1), resp["count"])
	patients := resp["patients"].([]interface{})
	assert.Len(t, patients, 1)
}

func TestSearchPatient_HospitalIsolation(t *testing.T) {
	db := newTestDB(t)
	mock := &mockHIS{}
	r := setupRouter(db, testSecret, mock)

	// Create hospitals
	hospA := model.Hospital{Name: "Mission Hospital"}
	hospB := model.Hospital{Name: "Bangkok Hospital"}
	require.NoError(t, db.Create(&hospA).Error)
	require.NoError(t, db.Create(&hospB).Error)

	// Seed patients
	patientA := model.Patient{
		FirstNameEN: "Palatip",
		NationalID:  "1234567890321",
		HospitalID:  hospA.ID,
	}
	patientB := model.Patient{
		FirstNameEN: "Bobby",
		NationalID:  "3210987654321",
		HospitalID:  hospB.ID,
	}
	require.NoError(t, db.Create(&patientA).Error)
	require.NoError(t, db.Create(&patientB).Error)

	token := generateToken(t, testSecret, 1, hospA.ID)

	req := httptest.NewRequest(http.MethodGet, "/patient/search", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, float64(1), resp["count"])

	patients := resp["patients"].([]interface{})
	require.Len(t, patients, 1)

	p := patients[0].(map[string]interface{})
	assert.Equal(t, "Palatip", p["first_name_en"])
}

func TestSearchPatient_WithNationalIDCallsHIS(t *testing.T) {
	db := newTestDB(t)

	// Create hospital
	hospital := model.Hospital{Name: "Mission Hospital"}
	require.NoError(t, db.Create(&hospital).Error)

	hisResponse := &external.PatientResponse{
		FirstNameEN: "Pang",
		LastNameEN:  "Talpai",
		NationalID:  "987654321987",
	}

	mock := &mockHIS{response: hisResponse}
	r := setupRouter(db, testSecret, mock)

	token := generateToken(t, testSecret, 1, hospital.ID)

	body := `{"national_id":"987654321987"}`
	req := httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify HIS call
	assert.True(t, mock.called)
	assert.Equal(t, "987654321987", mock.calledID)

	// Verify DB + response
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, float64(1), resp["count"])

	patients := resp["patients"].([]interface{})
	require.Len(t, patients, 1)

	p := patients[0].(map[string]interface{})
	assert.Equal(t, "Pang", p["first_name_en"])
	assert.Equal(t, "987654321987", p["national_id"])
}
