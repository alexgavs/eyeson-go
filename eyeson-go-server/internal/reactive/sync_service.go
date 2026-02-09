// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Sync Service

package reactive

import (
	"context"
	"errors"
	"eyeson-go-server/internal/eyesont"
	"eyeson-go-server/internal/models"
	"log"
	"time"

	"github.com/reactivex/rxgo/v2"
)

var (
	ErrInvalidTask = errors.New("invalid task")
)

// SyncService provides reactive synchronization with upstream API
type SyncService struct {
	client     *eyesont.Client
	repository *SimRepository
	config     StreamConfig
}

// NewSyncService creates a new reactive sync service
func NewSyncService(client *eyesont.Client) *SyncService {
	return &SyncService{
		client:     client,
		repository: NewSimRepository(),
		config:     DefaultStreamConfig(),
	}
}

// SyncTask represents a synchronization task
type SyncTask struct {
	Type    string
	Payload interface{}
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Task    SyncTask
	Success bool
	Error   error
	Data    interface{}
}

// ProcessTaskStream processes sync tasks reactively
func (s *SyncService) ProcessTaskStream(ctx context.Context, tasks <-chan rxgo.Item) *Stream {
	resultCh := make(chan rxgo.Item, s.config.BufferSize)
	
	go func() {
		defer close(resultCh)
		
		stream := NewStream(ctx, tasks, s.config)
		
		// Process tasks with parallelism and retry
		stream.
			// Retry failed tasks
			Retry(s.config.RetryAttempts).
			// Transform tasks to results
			Map(func(item interface{}) interface{} {
				task, ok := item.(SyncTask)
				if !ok {
					return SyncResult{
						Success: false,
						Error:   ErrInvalidTask,
					}
				}
				
				result := s.processTask(ctx, task)
				return result
			}).
			Subscribe(ctx,
				func(result interface{}) {
					resultCh <- rxgo.Of(result)
				},
				func(err error) {
					log.Printf("[ReactiveSync] Error: %v", err)
					resultCh <- rxgo.Error(err)
				},
				func() {
					log.Println("[ReactiveSync] Processing complete")
				},
			)
	}()
	
	return NewStream(ctx, resultCh, s.config)
}

// processTask executes a single sync task
func (s *SyncService) processTask(ctx context.Context, task SyncTask) SyncResult {
	result := SyncResult{
		Task:    task,
		Success: false,
	}
	
	switch task.Type {
	case "FETCH_SIM":
		// Not implemented - would need to fetch single SIM by ICCID
		result.Error = errors.New("FETCH_SIM not implemented")
		return result
		
	case "FETCH_ALL":
		// Fetch all SIMs from upstream
		response, err := s.client.GetSims(0, 1000, nil, "", "")
		if err != nil {
			result.Error = err
			return result
		}
		
		result.Success = true
		result.Data = response.Data
		
	case "UPDATE_STATUS":
		// Not implemented - would need UpdateSIMStatus method in client		result.Error = errors.New("UPDATE_STATUS not implemented")
		return result
	}
	
	return result
}

// SyncAllSims performs full synchronization with upstream
func (s *SyncService) SyncAllSims(ctx context.Context) *Stream {
	taskCh := make(chan rxgo.Item, 1)
	
	go func() {
		taskCh <- rxgo.Of(SyncTask{
			Type:    "FETCH_ALL",
			Payload: nil,
		})
		close(taskCh)
	}()
	
	// Process the fetch task
	return s.ProcessTaskStream(ctx, taskCh).
		// Extract SIMs from result
		Map(func(item interface{}) interface{} {
			result, ok := item.(SyncResult)
			if !ok || !result.Success {
				return []models.Sim{}
			}
			
			sims, ok := result.Data.([]models.Sim)
			if !ok {
				return []models.Sim{}
			}
			
			return sims
		})
}

// PeriodicSync creates a stream that syncs periodically
func (s *SyncService) PeriodicSync(ctx context.Context, interval time.Duration) *Stream {
	resultCh := make(chan rxgo.Item, s.config.BufferSize)
	
	go func() {
		defer close(resultCh)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("[ReactiveSync] Starting periodic sync")
				
				// Create sync task
				taskCh := make(chan rxgo.Item, 1)
				taskCh <- rxgo.Of(SyncTask{
					Type:    "FETCH_ALL",
					Payload: nil,
				})
				close(taskCh)
				
				// Process and emit results
				s.ProcessTaskStream(ctx, taskCh).
					Subscribe(ctx,
						func(result interface{}) {
							resultCh <- rxgo.Of(result)
						},
						func(err error) {
							log.Printf("[ReactiveSync] Periodic sync error: %v", err)
							resultCh <- rxgo.Error(err)
						},
						nil,
					)
			}
		}
	}()
	
	return NewStream(ctx, resultCh, s.config)
}

// MonitorChanges watches for changes and syncs affected SIMs
func (s *SyncService) MonitorChanges(ctx context.Context) *Stream {
	resultCh := make(chan rxgo.Item, s.config.BufferSize)
	
	go func() {
		defer close(resultCh)
		
		// Watch for SIM changes
		s.repository.WatchChanges(ctx, 5*time.Second).
			// Group changes by ICCID
			Buffer(5, 10*time.Second).
			// Create sync tasks for changed SIMs
			Map(func(batch interface{}) interface{} {
				items, ok := batch.([]interface{})
				if !ok {
					return []SyncTask{}
				}
				
				tasks := make([]SyncTask, 0, len(items))
				seen := make(map[string]bool)
				
				for _, item := range items {
					sim, ok := item.(models.Sim)
					if !ok || seen[sim.ICCID] {
						continue
					}
					
					tasks = append(tasks, SyncTask{
						Type:    "FETCH_SIM",
						Payload: sim.ICCID,
					})
					seen[sim.ICCID] = true
				}
				
				return tasks
			}).
			Subscribe(ctx,
				func(tasks interface{}) {
					taskList, ok := tasks.([]SyncTask)
					if !ok {
						return
					}
					
					// Create task stream
					taskCh := make(chan rxgo.Item, len(taskList))
					for _, task := range taskList {
						taskCh <- rxgo.Of(task)
					}
					close(taskCh)
					
					// Process tasks
					s.ProcessTaskStream(ctx, taskCh).
						Subscribe(ctx,
							func(result interface{}) {
								resultCh <- rxgo.Of(result)
							},
							func(err error) {
								resultCh <- rxgo.Error(err)
							},
							nil,
						)
				},
				func(err error) {
					resultCh <- rxgo.Error(err)
				},
				nil,
			)
	}()
	
	return NewStream(ctx, resultCh, s.config)
}
