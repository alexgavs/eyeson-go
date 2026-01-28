// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package eyesont

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"eyeson-go-server/internal/models"
)

// Instance - глобальный экземпляр клиента API
var Instance *Client

// Глобальный rate limiter для API запросов (WAF protection)
var apiRateMutex sync.Mutex
var lastApiCall time.Time

// Client представляет API-клиент EyesOnT с сессионной авторизацией
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	ApiDelayMs int
	httpClient *http.Client
	sessionMu  sync.RWMutex
	loggedIn   bool
	loginTime  time.Time
}

// Init инициализирует глобальный клиент API
func Init(cfg interface {
	GetApiBaseUrl() string
	GetApiUsername() string
	GetApiPassword() string
}) {
	// Используем рефлексию через структуру для совместимости
	type configLike interface {
		GetApiBaseUrl() string
		GetApiUsername() string
		GetApiPassword() string
	}

	// Попробуем прочитать напрямую из полей
	type configFields struct {
		ApiBaseUrl  string
		ApiUsername string
		ApiPassword string
	}

	// Проверим, поддерживает ли конфиг интерфейс
	if c, ok := cfg.(configLike); ok {
		Instance = NewClient(c.GetApiBaseUrl(), c.GetApiUsername(), c.GetApiPassword(), 1000)

		// Выполняем login при старте
		log.Println("[EyesOnT API] Performing initial startup login...")
		if err := Instance.Login(); err != nil {
			log.Printf("[EyesOnT API] WARNING: Initial login failed: %v", err)
		} else {
			log.Println("[EyesOnT API] Initial login successful")
		}
	}
}

// InitWithConfig инициализирует клиент с прямыми значениями
func InitWithConfig(baseURL, username, password string, apiDelayMs int) {
	Instance = NewClient(baseURL, username, password, apiDelayMs)
	maskedPassword := maskPassword(password)
	log.Printf("[EyesOnT API] Initialized: URL=%s, User=%s, Password=%s, Delay=%dms", baseURL, username, maskedPassword, apiDelayMs)

	// Выполняем login при старте
	log.Println("[EyesOnT API] Performing initial startup login...")
	if err := Instance.Login(); err != nil {
		log.Printf("[EyesOnT API] WARNING: Initial login failed: %v", err)
	} else {
		log.Println("[EyesOnT API] Initial login successful")
	}
}

// maskPassword скрывает пароль за звёздочками
func maskPassword(password string) string {
	if len(password) == 0 {
		return ""
	}
	if len(password) <= 2 {
		return "***"
	}
	return string(password[0]) + "***" + string(password[len(password)-1])
}

// maskPasswordInBody создаёт копию body со скрытым паролем для логирования
func maskPasswordInBody(body interface{}) interface{} {
	// Конвертируем в map для модификации
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return body
	}

	var m map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return body
	}

	// Маскируем пароль если есть
	if pwd, ok := m["password"].(string); ok {
		m["password"] = maskPassword(pwd)
	}

	return m
}

// NewClient создает новый клиент с cookie-jar для сессий
func NewClient(baseURL, username, password string, apiDelayMs int) *Client {
	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &Client{
		BaseURL:    baseURL,
		Username:   username,
		Password:   password,
		ApiDelayMs: apiDelayMs,
		httpClient: client,
		loggedIn:   false,
	}
}

// Login выполняет авторизацию и сохраняет сессионные cookies
func (c *Client) Login() error {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()

	loginReq := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: c.Username,
		Password: c.Password,
	}

	// Логируем попытку входа с маскированным паролем
	maskedReq := maskPasswordInBody(loginReq)
	if reqBytes, err := json.Marshal(maskedReq); err == nil {
		log.Printf("[EyesOnT API] AUTHORIZING USER: %s", string(reqBytes))
	}

	body, _ := json.Marshal(loginReq)
	url := fmt.Sprintf("%s/ipa/apis/json/general/login", c.BaseURL)

	log.Printf("[EyesOnT API] Sending LOGIN request to %s", url)

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[EyesOnT API] LOGIN RESPONSE (status=%d): %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != 200 {
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result    string `json:"result"`
		SessionId string `json:"sessionId"`
		JwtToken  string `json:"jwtToken"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Result == "SUCCESS" {
		c.loggedIn = true
		c.loginTime = time.Now()
		log.Printf("[EyesOnT API] LOGIN SUCCESS - SessionId: %s", result.SessionId)
		if len(result.JwtToken) > 20 {
			log.Printf("[EyesOnT API] JWT Token received (len=%d): %s...", len(result.JwtToken), result.JwtToken[:20])
		}
		return nil
	}

	// Даже если ответ не SUCCESS, cookies могут быть установлены
	c.loggedIn = true
	c.loginTime = time.Now()
	return nil
}

// EnsureSession проверяет и обновляет сессию при необходимости
func (c *Client) EnsureSession() error {
	c.sessionMu.RLock()
	needsLogin := !c.loggedIn || time.Since(c.loginTime) > 25*time.Minute
	c.sessionMu.RUnlock()

	if needsLogin {
		return c.Login()
	}
	return nil
}

// doRequest выполняет HTTP запрос с rate limiting для защиты от WAF
func (c *Client) doRequest(method, url string, body interface{}) (*http.Response, error) {
	// Rate limiting для защиты от WAF
	if c.ApiDelayMs > 0 {
		apiRateMutex.Lock()
		elapsed := time.Since(lastApiCall)
		delay := time.Duration(c.ApiDelayMs) * time.Millisecond
		if elapsed < delay {
			time.Sleep(delay - elapsed)
		}
		lastApiCall = time.Now()
		apiRateMutex.Unlock()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal error: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)

		// Логируем запрос (скрываем пароль)
		logBody := maskPasswordInBody(body)
		jsonIndented, _ := json.MarshalIndent(logBody, "", "  ")
		log.Printf("[EyesOnT API] Authenticated REQUEST to %s (with username/password in body)\nPayload: %s", url, string(jsonIndented))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// GetSims получает список SIM-карт
func (c *Client) GetSims(start, limit int, search []models.SearchParam, sortBy, sortDirection string) (*models.GetProvisioningDataResponse, error) {
	url := fmt.Sprintf("%s/ipa/apis/json/provisioning/getProvisioningData", c.BaseURL)

	// Формируем запрос с username и password
	type RequestBody struct {
		Start         int                  `json:"start"`
		Limit         int                  `json:"limit"`
		Search        []models.SearchParam `json:"search"`
		SortBy        string               `json:"sortBy"`
		SortDirection string               `json:"sortDirection"`
		Username      string               `json:"username"`
		Password      string               `json:"password"`
	}

	reqBody := RequestBody{
		Start:         start,
		Limit:         limit,
		Search:        search,
		SortBy:        sortBy,
		SortDirection: sortDirection,
		Username:      c.Username,
		Password:      c.Password,
	}

	resp, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Логируем ответ (сокращенно)
	respStr := string(body)
	if len(respStr) > 500 {
		respStr = respStr[:500] + "..."
	}
	log.Printf("[EyesOnT API] RESPONSE (status=%d):\n%s", resp.StatusCode, respStr)

	var result models.GetProvisioningDataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	log.Printf("[EyesOnT API] PARSED: result=%s, count=%d, dataLen=%d", result.Result, result.Count, len(result.Data))
	return &result, nil
}

// NormalizeMSISDN конвертирует 972xxx в 0xxx для Pelephone API
func NormalizeMSISDN(msisdn string) string {
	if strings.HasPrefix(msisdn, "972") && len(msisdn) == 12 {
		return "0" + msisdn[3:]
	}
	return msisdn
}

// BulkUpdate выполняет массовое обновление SIM-карт
// API формат: {"actions": [{"actionType": "...", "targetValue": "...", "subscribers": [{"neId": "..."}]}]}
func (c *Client) BulkUpdate(msisdns []string, actionType, targetValue string) (*models.BulkUpdateResponse, error) {
	url := fmt.Sprintf("%s/ipa/apis/json/provisioning/updateProvisioningData", c.BaseURL)

	// Нормализуем MSISDN для Pelephone API (972xxx -> 0xxx)
	subscribers := make([]map[string]string, len(msisdns))
	for i, m := range msisdns {
		subscribers[i] = map[string]string{"neId": NormalizeMSISDN(m)}
	}

	log.Printf("[EyesOnT API] BulkUpdate REQUEST: subscribers=%v (from %v), actionType=%s, targetValue=%s",
		subscribers, msisdns, actionType, targetValue)

	// Формируем запрос в правильном формате API
	type Action struct {
		ActionType  string              `json:"actionType"`
		TargetValue string              `json:"targetValue"`
		TargetId    string              `json:"targetId"`
		Subscribers []map[string]string `json:"subscribers"`
	}

	type RequestBody struct {
		Actions  []Action `json:"actions"`
		Username string   `json:"username"`
		Password string   `json:"password"`
	}

	reqBody := RequestBody{
		Actions: []Action{
			{
				ActionType:  actionType,
				TargetValue: targetValue,
				TargetId:    "",
				Subscribers: subscribers,
			},
		},
		Username: c.Username,
		Password: c.Password,
	}

	resp, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[EyesOnT API] BulkUpdate RESPONSE (status=%d): %s", resp.StatusCode, string(body))

	var result models.BulkUpdateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if result.Result == "SUCCESS" || result.Result == "succeeded" {
		log.Printf("[EyesOnT API] BulkUpdate SUCCESS: requestId=%d", result.RequestId)
	} else {
		log.Printf("[EyesOnT API] BulkUpdate FAILED: %s - %s", result.Result, result.Message)
	}

	return &result, nil
}

// GetJobs получает список задач провизионирования
func (c *Client) GetJobs(start, limit, jobId int, jobStatus string) (*models.GetJobsResponse, error) {
	url := fmt.Sprintf("%s/ipa/apis/json/provisioning/getProvisioningJobList", c.BaseURL)

	// Формируем запрос с username и password
	type RequestBody struct {
		Start     int    `json:"start,omitempty"`
		Limit     int    `json:"limit,omitempty"`
		JobId     int    `json:"jobId,omitempty"`
		JobStatus string `json:"jobStatus,omitempty"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}

	reqBody := RequestBody{
		Start:    start,
		Limit:    limit,
		Username: c.Username,
		Password: c.Password,
	}
	if jobId > 0 {
		reqBody.JobId = jobId
	}
	if jobStatus != "" {
		reqBody.JobStatus = jobStatus
	}

	resp, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	respStr := string(body)
	if len(respStr) > 500 {
		respStr = respStr[:500] + "..."
	}
	log.Printf("[EyesOnT API] GetJobs RESPONSE (status=%d): %s", resp.StatusCode, respStr)

	var result models.GetJobsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[EyesOnT API] GetJobs DECODE ERROR: %v", err)
		return nil, err
	}

	log.Printf("[EyesOnT API] GetJobs PARSED: result=%s, count=%d, jobsLen=%d", result.Result, result.Count, len(result.Jobs))
	return &result, nil
}

// CheckConnection checks if the API is reachable
func (c *Client) CheckConnection() bool {
	// Simple health check using login endpoint (assuming session is valid, or just checking reachability)
	// We can use a lightweight call, or just try to connect to base URL.
	// Since there isn't a dedicated "ping", we'll check if we can reach the server.

	resp, err := c.httpClient.Get(c.BaseURL + "/ipa/apis/json/general/login") // Method Not Allowed is fine, means we reached it.
	// Actually, our simulator only has POST.
	// Let's try to perform a dummy POST to login with bad credentials, or just check connectivity.
	// Or we can assume if the last request was < X seconds ago and successful, we are good.
	// But "real-time" check requires a request.

	// Better approach: Try a lightweight request or just a TCP dial?
	// Let's reuse doRequest for consistency (WAF handling etc), but with short timeout?
	// We'll just try to hit detailed status.

	// For simulator purposes, checking if we can get a response from any endpoint.
	// We will use a dummy request to /ipa/apis/json/general/login

	// NOTE: Since our simulator expects POST, a GET might return 404 or 405, but that implies connectivity = ONLINE.
	// Connection Refused means OFFLINE.

	req, _ := http.NewRequest("POST", c.BaseURL+"/ipa/apis/json/general/login", nil)
	// Short timeout for check
	client := *c.httpClient
	client.Timeout = 2 * time.Second

	resp, err = client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

// Close закрывает клиент
func (c *Client) Close() {
	// HTTP client не требует явного закрытия
}
