// +build ignore

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const (
	BaseURL  = "https://eot-portal.pelephone.co.il:8888"
	Username = "samsonixapi"
	Password = "pelephone@2020"
)

// Тестовые SIM карты (используем одни и те же для обоих тестов)
var testMSISDNs = []string{
	"972502686545",
	"972502686548",
	"972502686574",
	"972502686659",
	"972502686692",
}

type TestResult struct {
	Method       string
	TotalTime    time.Duration
	RequestCount int
	AllConfirmed bool
	Results      map[string]string
}

var httpClient *http.Client

func init() {
	jar, _ := cookiejar.New(nil)
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
}

func doRequest(method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}
	req, _ := http.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ============== ВАРИАНТ A: Polling по Job ID ==============
func testJobPolling() TestResult {
	result := TestResult{
		Method:  "Job ID Polling",
		Results: make(map[string]string),
	}
	start := time.Now()

	// 1. Получаем текущие статусы
	fmt.Println("\n=== ВАРИАНТ A: Job ID Polling ===")
	fmt.Println("1. Получаем текущие статусы...")
	
	currentStatuses := make(map[string]string)
	for _, msisdn := range testMSISDNs {
		status := getSingleSimStatus(msisdn)
		currentStatuses[msisdn] = status
		result.RequestCount++
		fmt.Printf("   %s: %s\n", msisdn, status)
	}

	// 2. Определяем целевой статус (toggle)
	targetStatus := "Suspended"
	if currentStatuses[testMSISDNs[0]] == "Suspended" {
		targetStatus = "Activated"
	}
	fmt.Printf("\n2. Меняем статус на: %s\n", targetStatus)

	// 3. Отправляем bulk update и получаем requestId
	requestId := sendBulkUpdate(testMSISDNs, targetStatus)
	result.RequestCount++
	fmt.Printf("   RequestId: %d\n", requestId)

	if requestId == 0 {
		fmt.Println("   ОШИБКА: не получен requestId")
		result.TotalTime = time.Since(start)
		return result
	}

	// 4. Polling по Job ID
	fmt.Println("\n3. Polling по Job ID...")
	maxAttempts := 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(3 * time.Second)
		
		jobStatus, actions := getJobStatus(requestId)
		result.RequestCount++
		
		fmt.Printf("   Attempt %d: Job status = %s\n", attempt, jobStatus)
		
		if jobStatus == "COMPLETED" || jobStatus == "SUCCESS" || jobStatus == "PARTIAL_SUCCESS" {
			fmt.Println("   ✓ Job завершён!")
			for msisdn, status := range actions {
				result.Results[msisdn] = status
			}
			result.AllConfirmed = true
			break
		}
		
		if jobStatus == "FAILED" {
			fmt.Println("   ✗ Job failed!")
			break
		}
	}

	result.TotalTime = time.Since(start)
	fmt.Printf("\n4. Результат: %d запросов за %v\n", result.RequestCount, result.TotalTime)
	return result
}

// ============== ВАРИАНТ B: Bulk GetSims Polling ==============
func testBulkSimsPolling() TestResult {
	result := TestResult{
		Method:  "Bulk GetSims Polling",
		Results: make(map[string]string),
	}
	start := time.Now()

	fmt.Println("\n=== ВАРИАНТ B: Bulk GetSims Polling ===")
	fmt.Println("1. Получаем текущие статусы (один запрос)...")

	// 1. Получаем все статусы одним запросом
	allStatuses := getBulkSimStatuses(testMSISDNs)
	result.RequestCount++
	
	for msisdn, status := range allStatuses {
		fmt.Printf("   %s: %s\n", msisdn, status)
	}

	// 2. Определяем целевой статус
	targetStatus := "Suspended"
	if allStatuses[testMSISDNs[0]] == "Suspended" {
		targetStatus = "Activated"
	}
	fmt.Printf("\n2. Меняем статус на: %s\n", targetStatus)

	// 3. Отправляем bulk update
	requestId := sendBulkUpdate(testMSISDNs, targetStatus)
	result.RequestCount++
	fmt.Printf("   RequestId: %d\n", requestId)

	// 4. Polling - один запрос на все SIM
	fmt.Println("\n3. Bulk polling статусов...")
	maxAttempts := 10
	pendingMSISDNs := make(map[string]bool)
	for _, m := range testMSISDNs {
		pendingMSISDNs[m] = true
	}

	for attempt := 1; attempt <= maxAttempts && len(pendingMSISDNs) > 0; attempt++ {
		time.Sleep(3 * time.Second)
		
		// Один запрос для всех pending
		currentStatuses := getBulkSimStatuses(testMSISDNs)
		result.RequestCount++
		
		confirmed := 0
		for msisdn := range pendingMSISDNs {
			if currentStatuses[msisdn] == targetStatus {
				result.Results[msisdn] = targetStatus
				delete(pendingMSISDNs, msisdn)
				confirmed++
			}
		}
		
		fmt.Printf("   Attempt %d: подтверждено %d, осталось %d\n", attempt, confirmed, len(pendingMSISDNs))
		
		if len(pendingMSISDNs) == 0 {
			result.AllConfirmed = true
			fmt.Println("   ✓ Все статусы подтверждены!")
			break
		}
	}

	result.TotalTime = time.Since(start)
	fmt.Printf("\n4. Результат: %d запросов за %v\n", result.RequestCount, result.TotalTime)
	return result
}

func getSingleSimStatus(msisdn string) string {
	url := fmt.Sprintf("%s/ipa/apis/json/provisioning/getProvisioningData", BaseURL)
	
	reqBody := map[string]interface{}{
		"username": Username,
		"password": Password,
		"start":    0,
		"limit":    1,
		"search": []map[string]string{
			{"fieldName": "MSISDN", "fieldValue": msisdn},
		},
	}
	
	resp, err := doRequest("POST", url, reqBody)
	if err != nil {
		return "ERROR"
	}
	
	var result struct {
		Data []struct {
			SimStatusChange string `json:"SIM_STATUS_CHANGE"`
		} `json:"data"`
	}
	json.Unmarshal(resp, &result)
	
	if len(result.Data) > 0 {
		return result.Data[0].SimStatusChange
	}
	return "NOT_FOUND"
}

func getBulkSimStatuses(msisdns []string) map[string]string {
	url := fmt.Sprintf("%s/ipa/apis/json/provisioning/getProvisioningData", BaseURL)
	
	// Загружаем все SIM (limit 5000) и фильтруем локально
	reqBody := map[string]interface{}{
		"username": Username,
		"password": Password,
		"start":    0,
		"limit":    5000,
	}
	
	resp, err := doRequest("POST", url, reqBody)
	if err != nil {
		return nil
	}
	
	var result struct {
		Data []struct {
			MSISDN          string `json:"MSISDN"`
			SimStatusChange string `json:"SIM_STATUS_CHANGE"`
		} `json:"data"`
	}
	json.Unmarshal(resp, &result)
	
	statuses := make(map[string]string)
	msisdnSet := make(map[string]bool)
	for _, m := range msisdns {
		msisdnSet[m] = true
	}
	
	for _, sim := range result.Data {
		if msisdnSet[sim.MSISDN] {
			statuses[sim.MSISDN] = sim.SimStatusChange
		}
	}
	
	return statuses
}

func sendBulkUpdate(msisdns []string, targetStatus string) int {
	url := fmt.Sprintf("%s/ipa/apis/json/provisioning/updateProvisioningData", BaseURL)
	
	subscribers := make([]map[string]string, len(msisdns))
	for i, m := range msisdns {
		// Normalize: 972xxx -> 0xxx
		neId := m
		if strings.HasPrefix(m, "972") && len(m) == 12 {
			neId = "0" + m[3:]
		}
		subscribers[i] = map[string]string{"neId": neId}
	}
	
	reqBody := map[string]interface{}{
		"username": Username,
		"password": Password,
		"actions": []map[string]interface{}{
			{
				"actionType":  "SIM_STATE_CHANGE",
				"targetValue": targetStatus,
				"subscribers": subscribers,
			},
		},
	}
	
	resp, err := doRequest("POST", url, reqBody)
	if err != nil {
		return 0
	}
	
	var result struct {
		Result    string `json:"result"`
		RequestId int    `json:"requestId"`
	}
	json.Unmarshal(resp, &result)
	
	return result.RequestId
}

func getJobStatus(jobId int) (string, map[string]string) {
	url := fmt.Sprintf("%s/ipa/apis/json/provisioning/getProvisioningJobList", BaseURL)
	
	reqBody := map[string]interface{}{
		"username": Username,
		"password": Password,
		"jobId":    jobId,
	}
	
	resp, err := doRequest("POST", url, reqBody)
	if err != nil {
		return "ERROR", nil
	}
	
	var result struct {
		Jobs []struct {
			JobStatus string `json:"jobStatus"`
			Actions   []struct {
				TargetValue string `json:"targetValue"`
				Status      string `json:"status"`
			} `json:"actions"`
		} `json:"jobs"`
	}
	json.Unmarshal(resp, &result)
	
	if len(result.Jobs) == 0 {
		return "NOT_FOUND", nil
	}
	
	job := result.Jobs[0]
	actions := make(map[string]string)
	for _, a := range job.Actions {
		actions[a.TargetValue] = a.Status
	}
	
	return job.JobStatus, actions
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ТЕСТ POLLING ВАРИАНТОВ A и B                         ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Тестовые SIM: %d шт                                          ║\n", len(testMSISDNs))
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// Тест A: Job ID polling
	resultA := testJobPolling()
	
	// Пауза между тестами
	fmt.Println("\n--- Пауза 10 сек между тестами ---")
	time.Sleep(10 * time.Second)
	
	// Тест B: Bulk GetSims polling  
	resultB := testBulkSimsPolling()

	// Сравнение результатов
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    СРАВНЕНИЕ РЕЗУЛЬТАТОВ                     ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Вариант A (Job ID):                                          ║\n")
	fmt.Printf("║   - Запросов: %d                                              ║\n", resultA.RequestCount)
	fmt.Printf("║   - Время: %v                                        ║\n", resultA.TotalTime.Round(time.Second))
	fmt.Printf("║   - Подтверждено: %v                                         ║\n", resultA.AllConfirmed)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Вариант B (Bulk GetSims):                                    ║\n")
	fmt.Printf("║   - Запросов: %d                                              ║\n", resultB.RequestCount)
	fmt.Printf("║   - Время: %v                                        ║\n", resultB.TotalTime.Round(time.Second))
	fmt.Printf("║   - Подтверждено: %v                                         ║\n", resultB.AllConfirmed)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	
	// Вывод победителя
	if resultA.RequestCount < resultB.RequestCount {
		fmt.Println("║ 🏆 ПОБЕДИТЕЛЬ: Вариант A (меньше запросов)                   ║")
	} else if resultB.RequestCount < resultA.RequestCount {
		fmt.Println("║ 🏆 ПОБЕДИТЕЛЬ: Вариант B (меньше запросов)                   ║")
	} else {
		fmt.Println("║ 🤝 НИЧЬЯ по количеству запросов                              ║")
	}
	
	if resultA.TotalTime < resultB.TotalTime {
		fmt.Println("║ ⏱️  Быстрее: Вариант A                                        ║")
	} else {
		fmt.Println("║ ⏱️  Быстрее: Вариант B                                        ║")
	}
	
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}
