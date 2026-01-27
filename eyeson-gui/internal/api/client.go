package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"eyeson-gui/internal/models"
)

type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HttpClient *http.Client
	SessionID  string
}

func NewClient(baseURL, username, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &Client{
		BaseURL:    baseURL,
		Username:   username,
		Password:   password,
		HttpClient: client,
	}, nil
}

func (c *Client) Login() error {
	endpoint := c.BaseURL + "/ipa/apis/json/general/login"
	log.Printf("[EyesOnGUI] Authenticating as %s to %s", c.Username, endpoint)

	reqBody := models.LoginRequest{
		Username: c.Username,
		Password: c.Password,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := c.HttpClient.Post(endpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[EyesOnGUI] Login failed with HTTP %d", resp.StatusCode)
		return fmt.Errorf("login failed: status code %d", resp.StatusCode)
	}

	var loginResp models.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return err
	}

	if loginResp.Result != "SUCCESS" {
		log.Printf("[EyesOnGUI] Login failed: %s - %s", loginResp.Result, loginResp.Message)
		return fmt.Errorf("login failed: %s - %s", loginResp.Result, loginResp.Message)
	}

	log.Printf("[EyesOnGUI] Login successful, SessionID acquired")
	c.SessionID = loginResp.SessionId
	return nil
}

func (c *Client) GetSims(start, limit int, search []models.SearchParam) (*models.GetProvisioningDataResponse, error) {
	endpoint := c.BaseURL + "/ipa/apis/json/provisioning/getProvisioningData"
	reqBody := models.GetProvisioningDataRequest{
		Username: c.Username,
		Password: c.Password,
		Start:    start,
		Limit:    limit,
		Search:   search,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.HttpClient.Post(endpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getSims failed: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var simResp models.GetProvisioningDataResponse
	if err := json.Unmarshal(bodyBytes, &simResp); err != nil {
		return nil, fmt.Errorf("decode error: %v, body: %s", err, string(bodyBytes))
	}

	if simResp.Result != "SUCCESS" {
		return nil, fmt.Errorf("api error: %s - %s", simResp.Result, simResp.Message)
	}

	return &simResp, nil
}

func (c *Client) BulkUpdateStatus(msisdns []string, newStatus string) (*models.UpdateProvisioningDataResponse, error) {
	endpoint := c.BaseURL + "/ipa/apis/json/provisioning/updateProvisioningData"

	var subscribers []models.SubscriberRequest
	for _, m := range msisdns {
		subscribers = append(subscribers, models.SubscriberRequest{NeId: m})
	}

	action := models.ProvisioningAction{
		ActionType:  "SIM_STATE_CHANGE",
		TargetValue: newStatus,
		Subscribers: subscribers,
	}

	reqBody := models.UpdateProvisioningDataRequest{
		Username: c.Username,
		Password: c.Password,
		Actions:  []models.ProvisioningAction{action},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.HttpClient.Post(endpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bulk update failed: status code %d", resp.StatusCode)
	}

	var updateResp models.UpdateProvisioningDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, err
	}

	if updateResp.Result != "SUCCESS" {
		return nil, fmt.Errorf("api error: %s - %s", updateResp.Result, updateResp.Message)
	}

	return &updateResp, nil
}
