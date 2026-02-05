// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Pelephone API Simulator

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	_ "github.com/mattn/go-sqlite3"
)

// SimCard represents a SIM card in the database
type SimCard struct {
	ID              int       `json:"id"`
	CLI             string    `json:"cli"`
	MSISDN          string    `json:"msisdn"`
	Status          string    `json:"sim_status_change"`
	RatePlan        string    `json:"rate_plan_full_name"`
	CustomerLabel1  string    `json:"customer_label_1"`
	CustomerLabel2  string    `json:"customer_label_2"`
	CustomerLabel3  string    `json:"customer_label_3"`
	SIMSwap         string    `json:"sim_swap"`
	IMSI            string    `json:"imsi"`
	IMEI            string    `json:"imei"`
	APNName         string    `json:"apn_name"`
	IP1             string    `json:"ip1"`
	MonthlyUsageMB  string    `json:"monthly_usage_mb"`
	AllocatedMB     string    `json:"allocated_mb"`
	LastSessionTime string    `json:"last_session_time"`
	InSession       string    `json:"in_session"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Config holds simulator configuration
type Config struct {
	SimCount int  `json:"sim_count"`
	Enabled  bool `json:"enabled"`
}

var db *sql.DB
var config Config

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	// Initialize database
	initDB()
	loadConfig()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// API Routes (Pelephone API simulation)
	app.Post("/ipa/apis/json/general/login", handleLogin)
	// Swagger: POST only
	app.Post("/ipa/apis/json/general/logout", handleLogout)
	app.Post("/ipa/apis/json/provisioning/getProvisioningData", handleGetProvisioningData)
	app.Post("/ipa/apis/json/provisioning/getProvisioningJobList", handleGetProvisioningJobList)
	app.Post("/ipa/apis/json/provisioning/getProvisioningParameterList", handleGetProvisioningParameterList)
	app.Post("/ipa/apis/json/provisioning/updateProvisioningData", handleUpdateProvisioningData)

	// Admin Panel Routes
	app.Get("/web/api/config", handleGetConfig)
	app.Post("/web/api/config", handleSetConfig)
	app.Get("/web/api/sims", handleGetSims)
	app.Post("/web/api/import", handleImportSims)
	app.Post("/web/api/generate", handleGenerateSims)
	app.Post("/web/api/clear", handleClearSims)
	app.Post("/web/api/sim/:cli/status", handleChangeSingleSimStatus)

	// Serve static files for admin panel
	app.Static("/web", "./web/static")

	// Default route redirect to admin panel
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/web")
	})

	log.Printf("========================================")
	log.Printf(" Pelephone API Simulator")
	log.Printf("========================================")
	log.Printf(" API Endpoint: http://localhost:%s", port)
	log.Printf(" Admin Panel:  http://localhost:%s/web", port)
	log.Printf("========================================")

	log.Fatal(app.Listen(":" + port))
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./simulator.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sim_cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cli TEXT UNIQUE NOT NULL,
			msisdn TEXT NOT NULL,
			status TEXT DEFAULT 'Activated',
			rate_plan TEXT DEFAULT '5GB Plan',
			customer_label_1 TEXT DEFAULT '',
			customer_label_2 TEXT DEFAULT '',
			customer_label_3 TEXT DEFAULT '',
			sim_swap TEXT DEFAULT '',
			imsi TEXT NOT NULL,
			imei TEXT DEFAULT '',
			apn_name TEXT DEFAULT 'internet.apn',
			ip1 TEXT DEFAULT '',
			monthly_usage_mb TEXT DEFAULT '0',
			allocated_mb TEXT DEFAULT '5120',
			last_session_time TEXT DEFAULT '',
			in_session TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT
		);

		CREATE TABLE IF NOT EXISTS provisioning_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			status TEXT NOT NULL,
			request_time INTEGER NOT NULL,
			last_action_time INTEGER NOT NULL,
			requested_application TEXT DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS provisioning_job_actions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL,
			ne_id TEXT NOT NULL,
			status TEXT NOT NULL,
			completion_time INTEGER NOT NULL,
			request_type TEXT NOT NULL,
			initial_value TEXT DEFAULT '',
			target_value TEXT DEFAULT '',
			error_msg TEXT DEFAULT '',
			error_desc TEXT DEFAULT '',
			FOREIGN KEY(job_id) REFERENCES provisioning_jobs(id)
		);
	`)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	log.Println("[Simulator] Database initialized")
}

func loadConfig() {
	config = Config{
		SimCount: 50,
		Enabled:  true,
	}

	row := db.QueryRow("SELECT value FROM config WHERE key = 'sim_count'")
	var val string
	if row.Scan(&val) == nil {
		if count, err := strconv.Atoi(val); err == nil {
			config.SimCount = count
		}
	}

	row = db.QueryRow("SELECT value FROM config WHERE key = 'enabled'")
	if row.Scan(&val) == nil {
		config.Enabled = val == "true"
	}

	// Check if we have SIMs, if not generate default
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sim_cards").Scan(&count)
	if count == 0 {
		generateSims(config.SimCount)
	}

	// Normalize legacy status values to match swagger enums.
	_, _ = db.Exec("UPDATE sim_cards SET status = 'Activated' WHERE status = 'Active'")
	_, _ = db.Exec("UPDATE sim_cards SET status = 'Pre-Activated' WHERE status = 'Pre-Active'")

	log.Printf("[Simulator] Config loaded: %d SIMs, enabled=%v", config.SimCount, config.Enabled)
}

func saveConfig() {
	db.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('sim_count', ?)", strconv.Itoa(config.SimCount))
	db.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('enabled', ?)", strconv.FormatBool(config.Enabled))
}

func generateSims(count int) {
	// Swagger enum: Activated, Pre-Activated, Suspended, Terminated
	statuses := []string{"Activated", "Suspended", "Terminated", "Pre-Activated"}
	ratePlans := []string{"5GB Plan", "10GB Plan", "20GB Plan", "Unlimited", "1GB Basic"}

	for i := 0; i < count; i++ {
		cli := fmt.Sprintf("05%08d", i)
		msisdn := cli
		imsi := fmt.Sprintf("42501%010d", i)
		status := statuses[rand.Intn(len(statuses))]
		ratePlan := ratePlans[rand.Intn(len(ratePlans))]
		ip := fmt.Sprintf("10.0.%d.%d", rand.Intn(256), rand.Intn(256))
		usage := strconv.Itoa(rand.Intn(5000))
		allocated := "5120"
		lastSession := time.Now().Add(-time.Duration(rand.Intn(720)) * time.Hour).Format("2006-01-02 15:04:05")
		label1 := fmt.Sprintf("Device %d", i)

		_, err := db.Exec(`
			INSERT OR REPLACE INTO sim_cards 
			(cli, msisdn, status, rate_plan, customer_label_1, imsi, apn_name, ip1, monthly_usage_mb, allocated_mb, last_session_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, cli, msisdn, status, ratePlan, label1, imsi, "internet.apn", ip, usage, allocated, lastSession)

		if err != nil {
			log.Printf("Error inserting SIM %s: %v", cli, err)
		}
	}

	log.Printf("[Simulator] Generated %d SIM cards", count)
}

// ========== Pelephone API Handlers ==========

func handleLogin(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result":  "FAILED",
			"message": "Simulator is disabled",
		})
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = c.BodyParser(&req)

	// Per PDF: unsuccessful example returns REJECTED with user fields.
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		return c.JSON(fiber.Map{
			"result":      "REJECTED",
			"message":     "Invalid Username or password",
			"userId":      0,
			"userType":    nil,
			"userGroupId": nil,
			"userLevel":   nil,
		})
	}

	log.Println("[Simulator] Login request received")
	// Swagger response shape: { result, sessionId, jwtToken }
	return c.JSON(fiber.Map{
		"result":    "SUCCESS",
		"sessionId": fmt.Sprintf("SIM-%d", time.Now().UnixNano()),
		"jwtToken":  fmt.Sprintf("SIMJWT-%d", rand.Int63()),
	})
}

func handleLogout(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result":  "FAILED",
			"message": "Simulator is disabled",
		})
	}

	log.Println("[Simulator] Logout request received")
	return c.JSON(fiber.Map{
		"result":  "SUCCESS",
		"message": nil,
	})
}

func handleGetProvisioningData(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result":  "FAILED",
			"message": "Simulator is disabled",
		})
	}

	// Match PDF schema.
	type SearchParam struct {
		FieldName  string `json:"fieldName"`
		FieldValue string `json:"fieldValue"`
	}
	var req struct {
		Username      string        `json:"username"`
		Password      string        `json:"password"`
		Start         int           `json:"start"`
		Limit         int           `json:"limit"`
		SortDirection string        `json:"sortDirection"`
		SortBy        string        `json:"sortBy"`
		Search        []SearchParam `json:"search"`
	}
	_ = c.BodyParser(&req)
	if req.Limit <= 0 {
		req.Limit = 500
	}
	if req.Start < 0 {
		req.Start = 0
	}

	// Build WHERE clause from search[] (PDF) with a safe whitelist.
	colMap := map[string]string{
		"CLI":                 "cli",
		"MSISDN":              "msisdn",
		"SIM_STATUS_CHANGE":   "status",
		"RATE_PLAN_FULL_NAME": "rate_plan",
		"CUSTOMER_LABEL_1":    "customer_label_1",
		"CUSTOMER_LABEL_2":    "customer_label_2",
		"CUSTOMER_LABEL_3":    "customer_label_3",
		"SIM_SWAP":            "sim_swap",
		"IMSI":                "imsi",
		"IMEI":                "imei",
		"APN_NAME":            "apn_name",
		"IP1":                 "ip1",
	}
	where := []string{"1=1"}
	args := []interface{}{}
	for _, s := range req.Search {
		fn := strings.TrimSpace(s.FieldName)
		fv := strings.TrimSpace(s.FieldValue)
		if fn == "" || fv == "" {
			continue
		}
		col, ok := colMap[fn]
		if !ok {
			continue
		}
		where = append(where, col+" LIKE ?")
		args = append(args, "%"+fv+"%")
	}
	whereSQL := strings.Join(where, " AND ")

	// Sorting
	sortCol := "cli"
	if strings.TrimSpace(req.SortBy) != "" {
		if col, ok := colMap[strings.TrimSpace(req.SortBy)]; ok {
			sortCol = col
		}
	}
	sortDir := "ASC"
	if strings.EqualFold(req.SortDirection, "DESC") {
		sortDir = "DESC"
	}

	var total int
	countSQL := "SELECT COUNT(*) FROM sim_cards WHERE " + whereSQL
	_ = db.QueryRow(countSQL, args...).Scan(&total)

	querySQL := fmt.Sprintf(`
		SELECT cli, msisdn, status, rate_plan, customer_label_1, customer_label_2, customer_label_3,
		       sim_swap, imsi, imei, apn_name, ip1, monthly_usage_mb, allocated_mb, last_session_time, in_session
		FROM sim_cards
		WHERE %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereSQL, sortCol, sortDir)
	args2 := append(append([]interface{}{}, args...), req.Limit, req.Start)
	rows, err := db.Query(querySQL, args2...)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"result": "FAILED", "message": err.Error()})
	}
	defer rows.Close()

	var sims []map[string]string
	for rows.Next() {
		var cli, msisdn, status, ratePlan, label1, label2, label3 string
		var simSwap, imsi, imei, apn, ip1, usage, allocated, lastSession, inSession string

		rows.Scan(&cli, &msisdn, &status, &ratePlan, &label1, &label2, &label3,
			&simSwap, &imsi, &imei, &apn, &ip1, &usage, &allocated, &lastSession, &inSession)

		prepaidBalance := allocated
		if prepaidBalance == "" {
			prepaidBalance = "0"
		}

		sims = append(sims, map[string]string{
			"CLI":                  cli,
			"MSISDN":               msisdn,
			"SIM_STATUS_CHANGE":    status,
			"RATE_PLAN_FULL_NAME":  ratePlan,
			"RATE_PLAN_CHANGE":     ratePlan,
			"CUSTOMER_LABEL_1":     label1,
			"CUSTOMER_LABEL_2":     label2,
			"CUSTOMER_LABEL_3":     label3,
			"SIM_SWAP":             simSwap,
			"IMSI":                 imsi,
			"IMEI":                 imei,
			"APN_NAME":             apn,
			"IP1":                  ip1,
			"MONTHLY_USAGE_MB":     usage,
			"ALLOCATED_MB":         allocated,
			"PREPAID_DATA_BALANCE": prepaidBalance,
			"LAST_SESSION_TIME":    lastSession,
			"IN_SESSION":           inSession,
		})
	}

	log.Printf("[Simulator] GetProvisioningData: returning %d SIMs (total: %d)", len(sims), total)

	fieldNames := []string{
		"CLI",
		"MSISDN",
		"SIM_STATUS_CHANGE",
		"RATE_PLAN_FULL_NAME",
		"CUSTOMER_LABEL_1",
		"CUSTOMER_LABEL_2",
		"CUSTOMER_LABEL_3",
		"SIM_SWAP",
		"IMSI",
		"IMEI",
		"APN_NAME",
		"IP1",
		"MONTHLY_USAGE_MB",
		"ALLOCATED_MB",
		"PREPAID_DATA_BALANCE",
		"LAST_SESSION_TIME",
		"IN_SESSION",
	}

	return c.JSON(fiber.Map{
		"result":     "SUCCESS",
		"message":    nil,
		"count":      total,
		"fieldNames": fieldNames,
		"data":       sims,
	})
}

func handleGetProvisioningJobList(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result":  "FAILED",
			"message": "Simulator is disabled",
		})
	}

	var req struct {
		Start     int    `json:"start"`
		Limit     int    `json:"limit"`
		JobId     int    `json:"jobId"`
		JobStatus string `json:"jobStatus"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}
	_ = c.BodyParser(&req)
	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.Start < 0 {
		req.Start = 0
	}

	where := []string{"1=1"}
	args := []interface{}{}
	if req.JobId > 0 {
		where = append(where, "id = ?")
		args = append(args, req.JobId)
	}
	if strings.TrimSpace(req.JobStatus) != "" {
		where = append(where, "status = ?")
		args = append(args, req.JobStatus)
	}
	whereSQL := strings.Join(where, " AND ")

	// Swagger does not specify sorting fields; keep deterministic order.
	sortCol := "id"
	sortDir := "DESC"

	var total int
	countSQL := "SELECT COUNT(*) FROM provisioning_jobs WHERE " + whereSQL
	_ = db.QueryRow(countSQL, args...).Scan(&total)

	querySQL := "SELECT id, status, request_time, last_action_time, requested_application FROM provisioning_jobs WHERE " + whereSQL + " ORDER BY " + sortCol + " " + sortDir + " LIMIT ? OFFSET ?"
	args2 := append(append([]interface{}{}, args...), req.Limit, req.Start)
	rows, err := db.Query(querySQL, args2...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"result": "FAILED", "message": err.Error()})
	}
	defer rows.Close()

	jobs := make([]fiber.Map, 0)
	for rows.Next() {
		var id int
		var status string
		var requestTime int64
		var lastActionTime int64
		var requestedApp string
		_ = rows.Scan(&id, &status, &requestTime, &lastActionTime, &requestedApp)

		// Load actions for job
		arows, aerr := db.Query(`
			SELECT ne_id, status, completion_time, request_type, initial_value, target_value, error_msg, error_desc
			FROM provisioning_job_actions
			WHERE job_id = ?
			ORDER BY id ASC
		`, id)
		actions := make([]fiber.Map, 0)
		if aerr == nil {
			for arows.Next() {
				var neID, aStatus, reqType, initialValue, targetValue, errorMsg, errorDesc string
				var completionTime int64
				_ = arows.Scan(&neID, &aStatus, &completionTime, &reqType, &initialValue, &targetValue, &errorMsg, &errorDesc)
				action := fiber.Map{
					"neId":        neID,
					"status":      aStatus,
					"actionType":  reqType,
					"targetValue": targetValue,
				}
				actions = append(actions, action)
			}
			arows.Close()
		}

		job := fiber.Map{
			"jobId":       id,
			"status":      status,
			"requestTime": requestTime,
			"actions":     actions,
		}
		jobs = append(jobs, job)
	}

	return c.JSON(fiber.Map{
		"result":  "SUCCESS",
		"message": nil,
		"count":   total,
		"jobs":    jobs,
	})
}

func handleGetProvisioningParameterList(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result":  "FAILED",
			"message": "Simulator is disabled",
		})
	}

	// Minimal but spec-shaped response.
	statuses := []string{"Activated", "Suspended", "Terminated", "Pre-Activated"}
	ratePlans := []string{"5GB Plan", "10GB Plan", "20GB Plan", "Unlimited", "1GB Basic"}

	toAvail := func(values []string) []fiber.Map {
		out := make([]fiber.Map, 0, len(values))
		for i, v := range values {
			out = append(out, fiber.Map{"value": i + 1, "name": v, "desc": v})
		}
		return out
	}

	parameters := []fiber.Map{
		{"fieldName": "SIM_STATUS_CHANGE", "alias": "SIM Status", "permissionLevel": "READ-WRITE_FROM_LIST", "availableValues": toAvail(statuses)},
		{"fieldName": "RATE_PLAN_FULL_NAME", "alias": "Rate Plan", "permissionLevel": "READ-WRITE_FROM_LIST", "availableValues": toAvail(ratePlans)},
		{"fieldName": "CUSTOMER_LABEL_1", "alias": "Customer Label 1", "permissionLevel": "READ-WRITE"},
		{"fieldName": "CUSTOMER_LABEL_2", "alias": "Customer Label 2", "permissionLevel": "READ-WRITE"},
		{"fieldName": "CUSTOMER_LABEL_3", "alias": "Customer Label 3", "permissionLevel": "READ-WRITE"},
		{"fieldName": "CLI", "alias": "CLI", "permissionLevel": "READ-ONLY"},
		{"fieldName": "MSISDN", "alias": "MSISDN", "permissionLevel": "READ-ONLY"},
		{"fieldName": "IMSI", "alias": "IMSI", "permissionLevel": "READ-ONLY"},
		{"fieldName": "APN_NAME", "alias": "APN", "permissionLevel": "READ-ONLY"},
	}

	return c.JSON(fiber.Map{
		"result":     "SUCCESS",
		"message":    nil,
		"parameters": parameters,
	})
}

func handleUpdateProvisioningData(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result":  "FAILED",
			"message": "Simulator is disabled",
		})
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Actions  []struct {
			ActionType  string `json:"actionType"`
			TargetValue string `json:"targetValue"`
			TargetId    string `json:"targetId"`
			Subscribers []struct {
				NeId string `json:"neId"`
			} `json:"subscribers"`
		} `json:"actions"`
	}

	// Use a raw decode first to help diagnose malformed JSON in logs.
	if len(c.Body()) > 0 {
		var raw any
		if err := json.Unmarshal(c.Body(), &raw); err != nil {
			return c.Status(400).JSON(fiber.Map{"result": "INVALID_REQ", "message": "Invalid JSON"})
		}
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"result": "INVALID_REQ", "message": "Invalid request"})
	}

	if len(req.Actions) == 0 {
		return c.Status(400).JSON(fiber.Map{"result": "INVALID_REQ", "message": "Missing actions"})
	}

	now := time.Now().Unix()
	res, err := db.Exec(
		"INSERT INTO provisioning_jobs(status, request_time, last_action_time, requested_application) VALUES(?, ?, ?, ?)",
		"COMPLETED",
		now,
		now,
		"Pelephone API Simulator",
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"result": "FAILED", "message": err.Error()})
	}
	jobID64, _ := res.LastInsertId()
	jobID := int(jobID64)

	anyFailure := false
	for _, action := range req.Actions {
		actionType := strings.TrimSpace(action.ActionType)
		if actionType == "" {
			anyFailure = true
			continue
		}

		for _, sub := range action.Subscribers {
			neID := strings.TrimSpace(sub.NeId)
			if neID == "" {
				anyFailure = true
				continue
			}

			// Find current values to populate "initialValue".
			var cli, status, ratePlan, label1, label2, label3 string
			row := db.QueryRow(
				"SELECT cli, status, rate_plan, customer_label_1, customer_label_2, customer_label_3 FROM sim_cards WHERE cli = ? OR msisdn = ? LIMIT 1",
				neID,
				neID,
			)
			scanErr := row.Scan(&cli, &status, &ratePlan, &label1, &label2, &label3)

			aStatus := "SUCCESS"
			errorDesc := ""
			errorMsg := ""
			initialValue := ""

			if scanErr != nil {
				aStatus = "FAILED"
				errorDesc = "SIM not found"
				errorMsg = "MISSING_ENTITY"
				anyFailure = true
			} else {
				// Apply the change.
				switch {
				case strings.EqualFold(actionType, "SIM_STATE_CHANGE") || strings.EqualFold(actionType, "SIM_STATUS_CHANGE"):
					initialValue = status
					_, err = db.Exec("UPDATE sim_cards SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE cli = ?", action.TargetValue, cli)
					if err != nil {
						aStatus = "FAILED"
						errorDesc = err.Error()
						errorMsg = "FAILED"
						anyFailure = true
					}
				case strings.EqualFold(actionType, "RATE_PLAN_CHANGE"):
					initialValue = ratePlan
					_, err = db.Exec("UPDATE sim_cards SET rate_plan = ?, updated_at = CURRENT_TIMESTAMP WHERE cli = ?", action.TargetValue, cli)
					if err != nil {
						aStatus = "FAILED"
						errorDesc = err.Error()
						errorMsg = "FAILED"
						anyFailure = true
					}
				case strings.HasPrefix(strings.ToUpper(actionType), "CUSTOMER_LABEL_"):
					upper := strings.ToUpper(actionType)
					col := "customer_label_1"
					switch upper {
					case "CUSTOMER_LABEL_1":
						initialValue = label1
						col = "customer_label_1"
					case "CUSTOMER_LABEL_2":
						initialValue = label2
						col = "customer_label_2"
					case "CUSTOMER_LABEL_3":
						initialValue = label3
						col = "customer_label_3"
					default:
						initialValue = label1
						col = "customer_label_1"
					}
					_, err = db.Exec("UPDATE sim_cards SET "+col+" = ?, updated_at = CURRENT_TIMESTAMP WHERE cli = ?", action.TargetValue, cli)
					if err != nil {
						aStatus = "FAILED"
						errorDesc = err.Error()
						errorMsg = "FAILED"
						anyFailure = true
					}
				case strings.EqualFold(actionType, "CUSTOMER_LABEL_UPDATE"):
					// Legacy client behavior: update label 1.
					initialValue = label1
					_, err = db.Exec("UPDATE sim_cards SET customer_label_1 = ?, updated_at = CURRENT_TIMESTAMP WHERE cli = ?", action.TargetValue, cli)
					if err != nil {
						aStatus = "FAILED"
						errorDesc = err.Error()
						errorMsg = "FAILED"
						anyFailure = true
					}
				default:
					aStatus = "REJECTED"
					errorDesc = "Unsupported actionType"
					errorMsg = actionType
					anyFailure = true
				}
			}

			_, _ = db.Exec(
				"INSERT INTO provisioning_job_actions(job_id, ne_id, status, completion_time, request_type, initial_value, target_value, error_msg, error_desc) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)",
				jobID,
				neID,
				aStatus,
				now,
				actionType,
				initialValue,
				action.TargetValue,
				errorMsg,
				errorDesc,
			)
		}
	}

	if anyFailure {
		// PDF examples show filtering by jobStatus="FAILED".
		_, _ = db.Exec("UPDATE provisioning_jobs SET status = ?, last_action_time = ? WHERE id = ?", "FAILED", now, jobID)
	}

	log.Printf("[Simulator] UpdateProvisioningData: job=%d actions=%d", jobID, len(req.Actions))
	return c.JSON(fiber.Map{
		"result":    "SUCCESS",
		"message":   nil,
		"requestId": jobID,
	})
}

func handleUpdateSIMStatus(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result":  "FAILED",
			"message": "Simulator is disabled",
		})
	}

	var req struct {
		CLI    string `json:"cli"`
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"result": "INVALID_REQ", "message": "Invalid request"})
	}

	result, err := db.Exec("UPDATE sim_cards SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE cli = ?", req.Status, req.CLI)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"result": "FAILED", "message": err.Error()})
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return c.Status(404).JSON(fiber.Map{"result": "MISSING_ENTITY", "message": "SIM not found"})
	}

	log.Printf("[Simulator] Updated SIM %s status to %s", req.CLI, req.Status)

	return c.JSON(fiber.Map{
		"result":  "SUCCESS",
		"message": nil,
	})
}

// ========== Admin Panel Handlers ==========

func handleGetConfig(c *fiber.Ctx) error {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sim_cards").Scan(&count)

	return c.JSON(fiber.Map{
		"sim_count":    config.SimCount,
		"enabled":      config.Enabled,
		"actual_count": count,
	})
}

func handleSetConfig(c *fiber.Ctx) error {
	var req Config
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	config.SimCount = req.SimCount
	config.Enabled = req.Enabled
	saveConfig()

	log.Printf("[Simulator] Config updated: sim_count=%d, enabled=%v", config.SimCount, config.Enabled)

	return c.JSON(fiber.Map{"success": true})
}

func handleGetSims(c *fiber.Ctx) error {
	rows, err := db.Query(`
		SELECT cli, msisdn, status, rate_plan, customer_label_1, imsi, ip1, monthly_usage_mb, last_session_time
		FROM sim_cards ORDER BY cli LIMIT 100
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var sims []map[string]string
	for rows.Next() {
		var cli, msisdn, status, ratePlan, label1, imsi, ip1, usage, lastSession string
		rows.Scan(&cli, &msisdn, &status, &ratePlan, &label1, &imsi, &ip1, &usage, &lastSession)

		sims = append(sims, map[string]string{
			"cli":          cli,
			"msisdn":       msisdn,
			"status":       status,
			"rate_plan":    ratePlan,
			"label":        label1,
			"imsi":         imsi,
			"ip":           ip1,
			"usage_mb":     usage,
			"last_session": lastSession,
		})
	}

	return c.JSON(sims)
}

func handleImportSims(c *fiber.Ctx) error {
	type simImport struct {
		CLI         string  `json:"cli"`
		MSISDN      string  `json:"msisdn"`
		Status      string  `json:"status"`
		RatePlan    string  `json:"rate_plan"`
		Label1      string  `json:"label1"`
		Label2      string  `json:"label2"`
		Label3      string  `json:"label3"`
		ICCID       string  `json:"iccid"`
		IMSI        string  `json:"imsi"`
		IMEI        string  `json:"imei"`
		APN         string  `json:"apn"`
		IP          string  `json:"ip"`
		UsageMB     string  `json:"usage_mb"`
		AllocatedMB string  `json:"allocated_mb"`
		LastSession string  `json:"last_session"`
		InSession   bool    `json:"in_session"`
		ID          *int    `json:"id,omitempty"`
		UpdatedAt   *string `json:"updated_at,omitempty"`
	}

	var req struct {
		Replace bool        `json:"replace"`
		SIMs    []simImport `json:"sims"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if !req.Replace {
		// Default to replace semantics for safety and predictability.
		req.Replace = true
	}

	tx, err := db.Begin()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer func() { _ = tx.Rollback() }()

	if req.Replace {
		_, _ = tx.Exec("DELETE FROM provisioning_job_actions")
		_, _ = tx.Exec("DELETE FROM provisioning_jobs")
		if _, err := tx.Exec("DELETE FROM sim_cards"); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO sim_cards
		(cli, msisdn, status, rate_plan, customer_label_1, customer_label_2, customer_label_3, sim_swap, imsi, imei, apn_name, ip1, monthly_usage_mb, allocated_mb, last_session_time, in_session, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer stmt.Close()

	inserted := 0
	for _, s := range req.SIMs {
		if strings.TrimSpace(s.CLI) == "" || strings.TrimSpace(s.MSISDN) == "" {
			continue
		}
		inSession := "false"
		if s.InSession {
			inSession = "true"
		}
		if _, err := stmt.Exec(
			s.CLI,
			s.MSISDN,
			s.Status,
			s.RatePlan,
			s.Label1,
			s.Label2,
			s.Label3,
			s.ICCID,
			s.IMSI,
			s.IMEI,
			s.APN,
			s.IP,
			s.UsageMB,
			s.AllocatedMB,
			s.LastSession,
			inSession,
		); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	config.SimCount = inserted
	saveConfig()

	log.Printf("[Simulator] Imported %d SIM cards (replace=%v)", inserted, req.Replace)
	return c.JSON(fiber.Map{"success": true, "count": inserted})
}

func handleGenerateSims(c *fiber.Ctx) error {
	var req struct {
		Count int `json:"count"`
	}
	if err := c.BodyParser(&req); err != nil || req.Count <= 0 {
		req.Count = config.SimCount
	}

	// Clear existing
	db.Exec("DELETE FROM sim_cards")

	// Generate new
	generateSims(req.Count)
	config.SimCount = req.Count
	saveConfig()

	return c.JSON(fiber.Map{
		"success": true,
		"count":   req.Count,
	})
}

func handleClearSims(c *fiber.Ctx) error {
	db.Exec("DELETE FROM sim_cards")
	log.Println("[Simulator] All SIM cards cleared")
	return c.JSON(fiber.Map{"success": true})
}

func handleChangeSingleSimStatus(c *fiber.Ctx) error {
	cli := c.Params("cli")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	result, err := db.Exec("UPDATE sim_cards SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE cli = ?", req.Status, cli)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "SIM not found"})
	}

	log.Printf("[Simulator] Admin changed SIM %s status to %s", cli, req.Status)
	return c.JSON(fiber.Map{"success": true})
}
