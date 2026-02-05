// Quick Pelephone API auth test
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	baseURL := "https://eot-portal.pelephone.co.il:8888"
	username := os.Getenv("EYESON_API_USERNAME")
	password := os.Getenv("EYESON_API_PASSWORD")

	fmt.Println("=== Pelephone API Full Test ===")
	fmt.Printf("URL: %s\n", baseURL)
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Password length: %d bytes\n", len(password))
	fmt.Println()

	// Create client with cookies
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
		},
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	// ========== LOGIN ==========
	loginReq := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(loginReq)
	loginURL := baseURL + "/ipa/apis/json/general/login"

	fmt.Printf("[1] POST %s\n", loginURL)

	req, _ := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
	setHeaders(req, baseURL)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("    Status: %d, Body: %s\n", resp.StatusCode, truncate(string(respBody), 200))

	var loginResult struct {
		Result   string `json:"result"`
		JwtToken string `json:"jwtToken"`
	}
	json.Unmarshal(respBody, &loginResult)
	if loginResult.Result != "SUCCESS" {
		fmt.Println("\n❌ LOGIN FAILED!")
		os.Exit(1)
	}
	fmt.Println("    ✅ LOGIN SUCCESS")
	fmt.Printf("    JWT Token: %s...\n", truncate(loginResult.JwtToken, 50))

	// Добавляем задержку как в браузере
	fmt.Println("    Waiting 2 seconds before next request...")
	time.Sleep(2 * time.Second)

	// ========== GET PROVISIONING DATA ==========
	fmt.Println()
	getSimsURL := baseURL + "/ipa/apis/json/provisioning/getProvisioningData"

	// Формат точно по swagger: RequestBase + start/limit/sortBy/sortDirection/search
	simsReq := map[string]interface{}{
		"username":      username,
		"password":      password,
		"start":         0,
		"limit":         50, // default по swagger
		"sortBy":        "",
		"sortDirection": "ASC",                 // enum: ASC or DESC
		"search":        []map[string]string{}, // массив объектов с fieldName/fieldValue
	}
	body, _ = json.Marshal(simsReq)

	fmt.Printf("[2] POST %s\n", getSimsURL)

	req, _ = http.NewRequest("POST", getSimsURL, bytes.NewBuffer(body))
	setHeaders(req, baseURL)
	// Добавляем JWT token в заголовок
	if loginResult.JwtToken != "" {
		req.Header.Set("Authorization", "Bearer "+loginResult.JwtToken)
	}

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	respBody, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("    Status: %d\n", resp.StatusCode)

	// Check if HTML (WAF block)
	bodyStr := string(respBody)
	if len(bodyStr) > 0 && bodyStr[0] == '<' {
		fmt.Printf("    ❌ WAF BLOCKED! Response:\n%s\n", truncate(bodyStr, 500))
		os.Exit(1)
	}

	fmt.Printf("    Body: %s\n", truncate(bodyStr, 300))

	var simsResult struct {
		Result string `json:"result"`
		Count  int    `json:"count"`
	}
	json.Unmarshal(respBody, &simsResult)
	fmt.Printf("    Result: %s, Count: %d\n", simsResult.Result, simsResult.Count)

	if simsResult.Result == "SUCCESS" {
		fmt.Println("\n✅ FULL API TEST PASSED!")
	} else {
		fmt.Println("\n❌ GetSims failed")
	}
}

func setHeaders(req *http.Request, baseURL string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,he;q=0.8")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
