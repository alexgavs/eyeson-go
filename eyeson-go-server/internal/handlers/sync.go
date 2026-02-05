package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/models"
	"eyeson-go-server/internal/services"
	"eyeson-go-server/internal/syncer"

	"github.com/gofiber/fiber/v2"
)

type manualSyncState struct {
	Running        bool      `json:"running"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	LastSuccess    bool      `json:"last_success"`
	LastError      string    `json:"last_error,omitempty"`
	LastProcessed  int       `json:"last_processed"`
	LastDurationMs int64     `json:"last_duration_ms"`
	Source         string    `json:"source"`
	BaseURL        string    `json:"base_url"`

	ClearLocalDB      bool `json:"clear_local_db"`
	DeletedBeforeSync int  `json:"deleted_before_sync"`

	SimulatorRequested  bool   `json:"simulator_requested"`
	SimulatorBaseURL    string `json:"simulator_base_url,omitempty"`
	SimulatorLastPushOK bool   `json:"simulator_last_push_ok"`
	SimulatorLastError  string `json:"simulator_last_error,omitempty"`
	SimulatorLastPushed int    `json:"simulator_last_pushed"`
	SimulatorDurationMs int64  `json:"simulator_duration_ms"`
}

var (
	manualSyncMu    sync.Mutex
	manualSyncStats = manualSyncState{Source: "pelephone"}
)

func GetManualSyncStatus(c *fiber.Ctx) error {
	manualSyncMu.Lock()
	st := manualSyncStats
	manualSyncMu.Unlock()
	return c.JSON(st)
}

type simulatorImportItem struct {
	CLI         string `json:"cli"`
	MSISDN      string `json:"msisdn"`
	Status      string `json:"status"`
	RatePlan    string `json:"rate_plan"`
	Label1      string `json:"label1"`
	Label2      string `json:"label2"`
	Label3      string `json:"label3"`
	ICCID       string `json:"iccid"`
	IMSI        string `json:"imsi"`
	IMEI        string `json:"imei"`
	APN         string `json:"apn"`
	IP          string `json:"ip"`
	UsageMB     string `json:"usage_mb"`
	AllocatedMB string `json:"allocated_mb"`
	LastSession string `json:"last_session"`
	InSession   bool   `json:"in_session"`
}

func pushLocalDBToSimulator(simBaseURL string) (int, error) {
	simBaseURL = strings.TrimRight(strings.TrimSpace(simBaseURL), "/")
	if simBaseURL == "" {
		return 0, fmt.Errorf("simulator base url is empty")
	}

	// Snapshot current local DB state.
	var sims []models.SimCard
	if err := database.DB.Find(&sims).Error; err != nil {
		return 0, err
	}

	items := make([]simulatorImportItem, 0, len(sims))
	for _, s := range sims {
		lastSession := ""
		if !s.LastSession.IsZero() {
			lastSession = s.LastSession.Format("2006-01-02 15:04:05")
		}
		items = append(items, simulatorImportItem{
			CLI:         s.CLI,
			MSISDN:      s.MSISDN,
			Status:      s.Status,
			RatePlan:    s.RatePlan,
			Label1:      s.Label1,
			Label2:      s.Label2,
			Label3:      s.Label3,
			ICCID:       s.ICCID,
			IMSI:        s.IMSI,
			IMEI:        s.IMEI,
			APN:         s.APN,
			IP:          s.IP,
			UsageMB:     strconv.FormatFloat(s.UsageMB, 'f', -1, 64),
			AllocatedMB: strconv.FormatFloat(s.AllocatedMB, 'f', -1, 64),
			LastSession: lastSession,
			InSession:   s.InSession,
		})
	}

	payload := struct {
		Replace bool                  `json:"replace"`
		SIMs    []simulatorImportItem `json:"sims"`
	}{
		Replace: true,
		SIMs:    items,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	url := simBaseURL + "/web/api/import"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody bytes.Buffer
		_, _ = errBody.ReadFrom(resp.Body)
		return 0, fmt.Errorf("simulator import failed (status=%d, body=%s)", resp.StatusCode, errBody.String())
	}

	var parsed struct {
		Success bool `json:"success"`
		Count   int  `json:"count"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	if parsed.Count == 0 {
		return len(items), nil
	}
	return parsed.Count, nil
}

// TriggerManualFullSync starts a background full sync into local DB.
// Source-of-truth is always Pelephone (EYESON_API_BASE_URL), regardless of selected upstream.
func TriggerManualFullSync(c *fiber.Ctx) error {
	var req struct {
		PushSimulator bool `json:"push_simulator"`
		ClearLocalDB  bool `json:"clear_local_db"`
	}
	_ = c.BodyParser(&req)
	pushSimulator := req.PushSimulator || c.QueryBool("push_simulator", false)
	clearLocalDB := req.ClearLocalDB || c.QueryBool("clear_local_db", false)

	manualSyncMu.Lock()
	if manualSyncStats.Running {
		st := manualSyncStats
		manualSyncMu.Unlock()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":  "sync already running",
			"status": st,
		})
	}
	manualSyncStats.Running = true
	manualSyncStats.StartedAt = time.Now()
	manualSyncStats.FinishedAt = time.Time{}
	manualSyncStats.LastError = ""
	manualSyncStats.LastProcessed = 0
	manualSyncStats.LastDurationMs = 0
	manualSyncStats.ClearLocalDB = clearLocalDB
	manualSyncStats.DeletedBeforeSync = 0
	manualSyncStats.SimulatorRequested = pushSimulator
	manualSyncStats.SimulatorLastError = ""
	manualSyncStats.SimulatorLastPushed = 0
	manualSyncStats.SimulatorLastPushOK = false
	manualSyncStats.SimulatorDurationMs = 0
	manualSyncMu.Unlock()

	go func() {
		start := time.Now()
		processed := 0
		deletedCount := 0
		var err error
		baseURL := ""
		simPushCount := 0
		simPushErr := ""
		simPushOK := false
		simPushDur := int64(0)
		simBase := ""

		cfg, cfgErr := config.LoadConfig()
		if cfgErr != nil {
			err = cfgErr
		} else {
			// Clear local DB if requested
			if clearLocalDB {
				result := database.DB.Exec("DELETE FROM sim_cards")
				if result.Error != nil {
					err = fmt.Errorf("failed to clear local DB: %w", result.Error)
				} else {
					deletedCount = int(result.RowsAffected)
					log.Printf("[ManualSync] Cleared local DB: deleted %d SIM records", deletedCount)
				}
			}

			if err == nil {
				// Always use hardcoded Pelephone URL for manual sync (source of truth)
				baseURL = services.DefaultPelephoneBaseURL
				client := eyesont.NewClient(baseURL, cfg.ApiUsername, cfg.ApiPassword, cfg.ApiDelayMs, cfg.ApiInsecureTLS)
				if loginErr := client.Login(); loginErr != nil {
					err = loginErr
				} else {
					s := &syncer.Syncer{DB: database.DB, Client: client}
					processed, err = s.SyncFull()

					if err == nil && pushSimulator {
						simBase = cfg.SimulatorBaseUrl
						pStart := time.Now()
						count, pErr := pushLocalDBToSimulator(simBase)
						simPushDur = time.Since(pStart).Milliseconds()
						simPushCount = count
						if pErr != nil {
							simPushErr = pErr.Error()
							err = fmt.Errorf("simulator push failed: %w", pErr)
						} else {
							simPushOK = true
						}
					}
				}
			}
		}

		dur := time.Since(start).Milliseconds()
		manualSyncMu.Lock()
		manualSyncStats.Running = false
		manualSyncStats.FinishedAt = time.Now()
		manualSyncStats.LastDurationMs = dur
		manualSyncStats.LastProcessed = processed
		manualSyncStats.DeletedBeforeSync = deletedCount
		manualSyncStats.BaseURL = baseURL
		manualSyncStats.SimulatorBaseURL = simBase
		manualSyncStats.SimulatorLastPushed = simPushCount
		manualSyncStats.SimulatorLastError = simPushErr
		manualSyncStats.SimulatorLastPushOK = simPushOK
		manualSyncStats.SimulatorDurationMs = simPushDur
		if err != nil {
			manualSyncStats.LastSuccess = false
			manualSyncStats.LastError = err.Error()
		} else {
			manualSyncStats.LastSuccess = true
			manualSyncStats.LastError = ""
		}
		manualSyncMu.Unlock()
	}()

	return c.JSON(fiber.Map{
		"started": true,
	})
}
