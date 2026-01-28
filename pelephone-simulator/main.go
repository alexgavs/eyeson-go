package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
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
	app.Post("/ipa/apis/json/provisioning/getProvisioningData", handleGetProvisioningData)
	app.Post("/ipa/apis/json/provisioning/updateSIMStatusChange", handleUpdateSIMStatus)

	// Admin Panel Routes
	app.Get("/web/api/config", handleGetConfig)
	app.Post("/web/api/config", handleSetConfig)
	app.Get("/web/api/sims", handleGetSims)
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
			status TEXT DEFAULT 'Active',
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

	log.Printf("[Simulator] Config loaded: %d SIMs, enabled=%v", config.SimCount, config.Enabled)
}

func saveConfig() {
	db.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('sim_count', ?)", strconv.Itoa(config.SimCount))
	db.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('enabled', ?)", strconv.FormatBool(config.Enabled))
}

func generateSims(count int) {
	statuses := []string{"Active", "Suspended", "Terminated", "Pre-Active"}
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
			"result": "failed",
			"error":  "Simulator is disabled",
		})
	}

	log.Println("[Simulator] Login request received")
	return c.JSON(fiber.Map{
		"result":    "succeeded",
		"sessionId": fmt.Sprintf("mock-session-%d", time.Now().Unix()),
	})
}

func handleGetProvisioningData(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result": "failed",
			"error":  "Simulator is disabled",
		})
	}

	var req struct {
		Start  int    `json:"start"`
		Limit  int    `json:"limit"`
		Search string `json:"search"`
	}
	if err := c.BodyParser(&req); err != nil {
		req.Limit = 500
	}
	if req.Limit == 0 {
		req.Limit = 500
	}

	rows, err := db.Query(`
		SELECT cli, msisdn, status, rate_plan, customer_label_1, customer_label_2, customer_label_3,
		       sim_swap, imsi, imei, apn_name, ip1, monthly_usage_mb, allocated_mb, last_session_time, in_session
		FROM sim_cards
		ORDER BY cli
		LIMIT ? OFFSET ?
	`, req.Limit, req.Start)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"result": "failed", "error": err.Error()})
	}
	defer rows.Close()

	var sims []map[string]string
	for rows.Next() {
		var cli, msisdn, status, ratePlan, label1, label2, label3 string
		var simSwap, imsi, imei, apn, ip1, usage, allocated, lastSession, inSession string

		rows.Scan(&cli, &msisdn, &status, &ratePlan, &label1, &label2, &label3,
			&simSwap, &imsi, &imei, &apn, &ip1, &usage, &allocated, &lastSession, &inSession)

		sims = append(sims, map[string]string{
			"CLI":                 cli,
			"MSISDN":              msisdn,
			"SIM_STATUS_CHANGE":   status,
			"RATE_PLAN_FULL_NAME": ratePlan,
			"CUSTOMER_LABEL_1":    label1,
			"CUSTOMER_LABEL_2":    label2,
			"CUSTOMER_LABEL_3":    label3,
			"SIM_SWAP":            simSwap,
			"IMSI":                imsi,
			"IMEI":                imei,
			"APN_NAME":            apn,
			"IP1":                 ip1,
			"MONTHLY_USAGE_MB":    usage,
			"ALLOCATED_MB":        allocated,
			"LAST_SESSION_TIME":   lastSession,
			"IN_SESSION":          inSession,
		})
	}

	var total int
	db.QueryRow("SELECT COUNT(*) FROM sim_cards").Scan(&total)

	log.Printf("[Simulator] GetProvisioningData: returning %d SIMs (total: %d)", len(sims), total)

	return c.JSON(fiber.Map{
		"result": "succeeded",
		"count":  total,
		"data":   sims,
	})
}

func handleUpdateSIMStatus(c *fiber.Ctx) error {
	if !config.Enabled {
		return c.Status(503).JSON(fiber.Map{
			"result": "failed",
			"error":  "Simulator is disabled",
		})
	}

	var req struct {
		CLI    string `json:"cli"`
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"result": "failed", "error": "Invalid request"})
	}

	result, err := db.Exec("UPDATE sim_cards SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE cli = ?", req.Status, req.CLI)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"result": "failed", "error": err.Error()})
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return c.Status(404).JSON(fiber.Map{"result": "failed", "error": "SIM not found"})
	}

	log.Printf("[Simulator] Updated SIM %s status to %s", req.CLI, req.Status)

	return c.JSON(fiber.Map{
		"result": "succeeded",
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
