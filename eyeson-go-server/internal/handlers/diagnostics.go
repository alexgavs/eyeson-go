// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

type endpointProbeResult struct {
	URL        string `json:"url"`
	Method     string `json:"method"`
	StatusCode int    `json:"status_code"`
	OK         bool   `json:"ok"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type simulatorConfig struct {
	SimCount    int  `json:"sim_count"`
	Enabled     bool `json:"enabled"`
	ActualCount int  `json:"actual_count"`
}

func probeJSON(client *http.Client, method, url string, payload any) endpointProbeResult {
	res := endpointProbeResult{URL: url, Method: method}

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			res.Error = fmt.Sprintf("marshal: %v", err)
			return res
		}
		body = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := client.Do(req)
	res.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	res.StatusCode = resp.StatusCode

	// Capture small body for troubleshooting.
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyStr := strings.TrimSpace(string(bodyBytes))

	// Fiber's default 404/405 for missing routes typically looks like: "Cannot POST /path".
	missingRoute := false
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		cannotPrefix := "Cannot " + strings.ToUpper(method) + " "
		missingRoute = strings.Contains(bodyStr, cannotPrefix) || strings.Contains(bodyStr, "Cannot "+strings.ToUpper(method))
	}

	// Endpoint exists if it's not a missing route. Note: some endpoints legitimately return 404 (e.g. SIM not found).
	res.OK = !missingRoute

	// Preserve body as error text for non-OK or non-2xx results.
	if bodyStr != "" {
		if !res.OK {
			res.Error = bodyStr
		} else if resp.StatusCode >= 400 {
			res.Error = bodyStr
		}
	}

	return res
}

// GetAPIDiagnostics returns provider endpoint reachability/capabilities for troubleshooting.
// GET /api/v1/api-status/diagnostics (Admin only)
func GetAPIDiagnostics(c *fiber.Ctx) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Could not load config: " + err.Error()})
	}

	selected, selErr := services.GetUpstreamSelected()
	if selErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Could not read upstream selection: " + selErr.Error()})
	}

	base := strings.TrimRight(services.ResolveUpstreamBaseURL(cfg, selected), "/")
	client := &http.Client{Timeout: 2 * time.Second}

	loginURL := base + "/ipa/apis/json/general/login"
	getSimsURL := base + "/ipa/apis/json/provisioning/getProvisioningData"
	updateProvisioningURL := base + "/ipa/apis/json/provisioning/updateProvisioningData"
	updateLegacyURL := base + "/ipa/apis/json/provisioning/updateSIMStatusChange"

	loginProbe := probeJSON(client, http.MethodPost, loginURL, map[string]string{
		"username": cfg.ApiUsername,
		"password": cfg.ApiPassword,
	})

	getSimsProbe := probeJSON(client, http.MethodPost, getSimsURL, fiber.Map{
		"start":         0,
		"limit":         1,
		"search":        []fiber.Map{},
		"sortBy":        "",
		"sortDirection": "",
		"username":      cfg.ApiUsername,
		"password":      cfg.ApiPassword,
	})

	// Safe "existence" probe: use a non-existing subscriber (0000000000) to avoid side effects.
	updateProvisioningProbe := probeJSON(client, http.MethodPost, updateProvisioningURL, fiber.Map{
		"username": cfg.ApiUsername,
		"password": cfg.ApiPassword,
		"actions": []fiber.Map{
			{
				"actionType":  "SIM_STATE_CHANGE",
				"targetValue": "Active",
				"targetId":    "",
				"subscribers": []fiber.Map{{"neId": "0000000000"}},
			},
		},
	})

	updateLegacyProbe := probeJSON(client, http.MethodPost, updateLegacyURL, map[string]string{
		"cli":    "0000000000",
		"status": "Active",
	})

	// Optional: if upstream is set to our simulator, read its admin config.
	var simCfg *simulatorConfig
	if selected == services.UpstreamSimulator {
		simURL := base + "/web/api/config"
		req, _ := http.NewRequest(http.MethodGet, simURL, nil)
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var parsed simulatorConfig
				if decErr := json.NewDecoder(resp.Body).Decode(&parsed); decErr == nil {
					simCfg = &parsed
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"upstream": fiber.Map{
			"selected": string(selected),
		},
		"provider": fiber.Map{
			"base_url": base,
			"endpoints": fiber.Map{
				"login":                  loginProbe,
				"getProvisioningData":    getSimsProbe,
				"updateProvisioningData": updateProvisioningProbe,
				"updateSIMStatusChange":  updateLegacyProbe,
			},
			"simulator": simCfg,
		},
		"server_time": time.Now().Format(time.RFC3339),
	})
}
