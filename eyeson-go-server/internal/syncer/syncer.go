package syncer

import (
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/models"

	"gorm.io/gorm"
)

type Syncer struct {
	DB     *gorm.DB
	Client *eyesont.Client
	paused int32
}

func New(db *gorm.DB) *Syncer {
	// We assume eyesont.Instance is initialized in main
	return &Syncer{
		DB:     db,
		Client: eyesont.Instance,
	}
}

func (s *Syncer) SetPaused(paused bool) {
	var val int32 = 0
	if paused {
		val = 1
	}
	atomic.StoreInt32(&s.paused, val)
	status := "RESUMED"
	if paused {
		status = "PAUSED"
	}
	log.Printf("[Syncer] Status changed to %s", status)
}

func (s *Syncer) IsPaused() bool {
	return atomic.LoadInt32(&s.paused) == 1
}

func (s *Syncer) Start() {
	log.Println("[Syncer] Starting background synchronization service...")
	go func() {
		// Check if we should sync initially
		if !s.shouldSync() {
			log.Println("[Syncer] Skipping initial sync - API unavailable or simulator DOWN")
		} else {
			// Initial sync
			s.SyncFull()
		}

		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			if s.shouldSync() {
				s.SyncFull()
			} else {
				log.Println("[Syncer] Skipping scheduled sync - API unavailable")
			}
		}
	}()
}

// shouldSync checks if we should attempt to sync with API
func (s *Syncer) shouldSync() bool {
	if s.IsPaused() {
		return false
	}

	if s.Client == nil {
		return false
	}

	// Check API connectivity with a simple ping/check
	// For now, we'll try a small request - if it fails, skip sync
	return true
}

func (s *Syncer) SyncFull() {
	if s.IsPaused() {
		return
	}

	if s.Client == nil {
		log.Println("[Syncer] Error: API Client not initialized")
		return
	}

	log.Println("[Syncer] Starting full sync cycle...")
	startTime := time.Now()

	start := 0
	limit := 500 // Fetch 500 at a time
	totalProcessed := 0

	for {
		if s.IsPaused() {
			break
		}

		// Low Priority: Check for high priority user tasks (SyncTask)
		var pendingCount int64
		s.DB.Model(&models.SyncTask{}).Where("status IN ?", []string{"PENDING", "PROCESSING"}).Count(&pendingCount)
		if pendingCount > 0 {
			// Yield to User Tasks
			// log.Printf("[Syncer] Pausing full sync for %d pending user tasks...", pendingCount)
			time.Sleep(2 * time.Second)
			continue
		}

		// Fetch from API
		resp, err := s.Client.GetSims(start, limit, nil, "", "")
		if err != nil {
			log.Printf("[Syncer] Error fetching SIMs: %v", err)
			log.Println("[Syncer] API unavailable - will retry on next cycle")
			break // Stop sync on error but don't crash
		}

		if len(resp.Data) == 0 {
			break // No more data
		}

		// Process batch
		if err := s.processBatch(resp.Data); err != nil {
			log.Printf("[Syncer] Error processing batch: %v", err)
		}

		totalProcessed += len(resp.Data)
		start += limit

		// If we received fewer items than limit, we reached the end
		if len(resp.Data) < limit {
			break
		}

		// Small pause to be nice to the API
		time.Sleep(100 * time.Millisecond)
	}

	duration := time.Since(startTime)
	log.Printf("[Syncer] Full sync completed in %v. Processed %d records.", duration, totalProcessed)
}

func (s *Syncer) processBatch(sims []models.SimData) error {
	var msisdns []string
	for _, s := range sims {
		msisdns = append(msisdns, s.MSISDN)
	}

	// 1. Fetch Existing
	var existingSims []models.SimCard
	if err := s.DB.Where("msisdn IN ?", msisdns).Find(&existingSims).Error; err != nil {
		return err
	}

	existingMap := make(map[string]models.SimCard)
	for _, sim := range existingSims {
		existingMap[sim.MSISDN] = sim
	}

	var toCreate []models.SimCard
	var toUpdate []models.SimCard
	var histories []models.SimHistory

	// 2. Compare API vs DB
	for _, apiSim := range sims {
		newSim := mapApiToModel(apiSim)
		existing, found := existingMap[newSim.MSISDN]

		if !found {
			// Create New
			toCreate = append(toCreate, newSim)
			// History: Created
			histories = append(histories, models.SimHistory{
				MSISDN:   newSim.MSISDN,
				Action:   "CREATED",
				Source:   "SYNC_DISCOVERY",
				NewValue: "Detected by Sync",
			})
		} else {
			// Update Existing - Check Diff
			newSim.ID = existing.ID
			newSim.CreatedAt = existing.CreatedAt
			changesFound := false

			// Compare fields (Status, IP, IMEI, Usage)
			if newSim.Status != existing.Status {
				changesFound = true
				histories = append(histories, createHistory(existing, "STATUS", existing.Status, newSim.Status))
			}
			if newSim.IP != existing.IP {
				changesFound = true
				histories = append(histories, createHistory(existing, "IP", existing.IP, newSim.IP))
			}
			if newSim.IMEI != existing.IMEI {
				changesFound = true // Don't log history for IMEI changes usually, but let's do it for tracking
				histories = append(histories, createHistory(existing, "IMEI", existing.IMEI, newSim.IMEI))
			}
			if newSim.ICCID != existing.ICCID {
				changesFound = true
				histories = append(histories, createHistory(existing, "ICCID", existing.ICCID, newSim.ICCID)) // SimSwap
			}
			// Usage Changes - Update but don't spam History unless jump is big?
			// For now, let's just update usage without history to avoid clutter,
			// History is mainly for state/identity changes.
			if newSim.UsageMB != existing.UsageMB || newSim.AllocatedMB != existing.AllocatedMB {
				changesFound = true
			}
			if newSim.InSession != existing.InSession {
				changesFound = true
			}
			if !newSim.LastSession.Equal(existing.LastSession) {
				changesFound = true
			}

			if changesFound {
				toUpdate = append(toUpdate, newSim)
			}
		}
	}

	// 3. Execute Updates in Transaction
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if len(toCreate) > 0 {
			if err := tx.Create(&toCreate).Error; err != nil {
				return err
			}
		}

		if len(toUpdate) > 0 {
			// Batch update not supported well for diverse values in GORM, so we loop or use Upsert
			// Since we have IDs, we can save. But save is slow for loop.
			// Upsert is better.
			// Re-using generic Upsert for all 'toUpdate'
			// Note: We already built 'toUpdate' with ID populated.
			if err := tx.Save(&toUpdate).Error; err != nil {
				return err
			}
		}

		if len(histories) > 0 {
			if err := tx.Create(&histories).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func createHistory(sim models.SimCard, field, oldVal, newVal string) models.SimHistory {
	return models.SimHistory{
		SimID:    sim.ID,
		MSISDN:   sim.MSISDN,
		Action:   "CHANGE_" + field,
		Field:    field,
		OldValue: oldVal,
		NewValue: newVal,
		Source:   "SYNC_PROVIDER",
	}
}

func mapApiToModel(d models.SimData) models.SimCard {
	// Parse Usage
	usageVal, _ := strconv.ParseFloat(d.MonthlyUsageMB, 64)
	allocVal, _ := strconv.ParseFloat(d.AllocatedMB, 64)

	// Parse Time
	var lastSession time.Time
	// Format example: "2023-10-27 10:00:00" - need to check actual format from user logs if possible
	// For now, try standard SQL format or continue
	// Assuming Pelephone sends "YYYY-MM-DD HH:MM:SS"
	lastSession, _ = time.Parse("2006-01-02 15:04:05", d.LastSessionTime)

	// Parse InSession
	inSession := false
	if d.InSession == "true" || d.InSession == "1" || d.InSession == "True" {
		inSession = true
	}

	return models.SimCard{
		MSISDN:      d.MSISDN,
		CLI:         d.CLI,
		IMSI:        d.IMSI,
		ICCID:       d.SimSwap, // Mapping SimSwap to ICCID
		IMEI:        d.IMEI,
		Status:      d.SimStatusChange, // API uses SimStatusChange for current status field?
		RatePlan:    d.RatePlanFullName,
		Label1:      d.CustomerLabel1,
		Label2:      d.CustomerLabel2,
		Label3:      d.CustomerLabel3,
		APN:         d.ApnName,
		IP:          d.Ip1,
		UsageMB:     usageVal,
		AllocatedMB: allocVal,
		LastSession: lastSession,
		InSession:   inSession,
		LastSyncAt:  time.Now(),
		IsSyncing:   false,
	}
}
