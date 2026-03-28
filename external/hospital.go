package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// HISSearcher is the interface for searching patients from an external HIS.
type HISSearcher interface {
	SearchPatient(ctx context.Context, id string) (*PatientResponse, error)
}

// PatientResponse maps external API response
type PatientResponse struct {
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`
	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	PatientHN    string `json:"patient_hn"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

// HISClient implements HISSearcher
type HISClient struct {
	BaseURL string
	Client  *http.Client
}

// NewHISClient creates a new HISClient with the given base URL.
func NewHISClient(baseURL string) *HISClient {
	return &HISClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}
}

// SearchPatient fetches patient from external API
func (c *HISClient) SearchPatient(ctx context.Context, id string) (*PatientResponse, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}

	escapedID := url.PathEscape(id)
	endpoint := fmt.Sprintf("%s/patient/search/%s", c.BaseURL, escapedID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call HIS API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("patient not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HIS API status: %d", resp.StatusCode)
	}

	var patient PatientResponse
	if err := json.NewDecoder(resp.Body).Decode(&patient); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &patient, nil
}
