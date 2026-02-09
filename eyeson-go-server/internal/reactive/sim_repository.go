// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Repository

package reactive

import (
	"context"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"
	"time"

	"github.com/reactivex/rxgo/v2"
)

// SimRepository provides reactive operations for SIM cards
type SimRepository struct {
	config StreamConfig
}

// NewSimRepository creates a new reactive SIM repository
func NewSimRepository() *SimRepository {
	return &SimRepository{
		config: DefaultStreamConfig(),
	}
}

// GetAllAsStream returns all SIMs as a reactive stream
func (r *SimRepository) GetAllAsStream(ctx context.Context) *Stream {
	ch := make(chan rxgo.Item)

	go func() {
		defer close(ch)

		var sims []models.Sim
		if err := database.DB.Find(&sims).Error; err != nil {
			ch <- rxgo.Error(err)
			return
		}

		for _, sim := range sims {
			select {
			case <-ctx.Done():
				return
			case ch <- rxgo.Of(sim):
			}
		}
	}()

	return NewStream(ctx, ch, r.config)
}

// WatchChanges creates a stream that emits on SIM changes
func (r *SimRepository) WatchChanges(ctx context.Context, pollInterval time.Duration) *Stream {
	ch := make(chan rxgo.Item)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		var lastUpdate time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var sims []models.Sim
				if err := database.DB.Where("updated_at > ?", lastUpdate).Find(&sims).Error; err != nil {
					ch <- rxgo.Error(err)
					continue
				}

				for _, sim := range sims {
					ch <- rxgo.Of(sim)
					if sim.UpdatedAt.After(lastUpdate) {
						lastUpdate = sim.UpdatedAt
					}
				}
			}
		}
	}()

	return NewStream(ctx, ch, r.config)
}

// FindByStatusStream returns SIMs filtered by status as a stream
func (r *SimRepository) FindByStatusStream(ctx context.Context, status string) *Stream {
	ch := make(chan rxgo.Item)

	go func() {
		defer close(ch)

		var sims []models.Sim
		if err := database.DB.Where("status = ?", status).Find(&sims).Error; err != nil {
			ch <- rxgo.Error(err)
			return
		}

		for _, sim := range sims {
			select {
			case <-ctx.Done():
				return
			case ch <- rxgo.Of(sim):
			}
		}
	}()

	return NewStream(ctx, ch, r.config)
}

// BatchUpdate applies updates to multiple SIMs reactively
func (r *SimRepository) BatchUpdate(ctx context.Context, updates <-chan rxgo.Item) *Stream {
	resultCh := make(chan rxgo.Item, r.config.BufferSize)

	go func() {
		defer close(resultCh)

		stream := NewStream(ctx, updates, r.config)

		// Buffer updates for batch processing
		stream.Buffer(10, 2*time.Second).
			Subscribe(ctx,
				func(batch interface{}) {
					items, ok := batch.([]interface{})
					if !ok {
						return
					}

					// Process batch
					for _, item := range items {
						update, ok := item.(models.SimUpdate)
						if !ok {
							continue
						}

						// Update in database
						if err := database.DB.Model(&models.Sim{}).
							Where("iccid = ?", update.SimSwap).
							Updates(update).Error; err != nil {
							resultCh <- rxgo.Error(err)
						} else {
							resultCh <- rxgo.Of(update)
						}
					}
				},
				func(err error) {
					resultCh <- rxgo.Error(err)
				},
				nil,
			)
	}()

	return NewStream(ctx, resultCh, r.config)
}

// SearchStream performs reactive search with debouncing
func (r *SimRepository) SearchStream(ctx context.Context, searchTerms <-chan rxgo.Item) *Stream {
	resultCh := make(chan rxgo.Item, r.config.BufferSize)

	go func() {
		defer close(resultCh)

		stream := NewStream(ctx, searchTerms, r.config)

		// Debounce search requests
		stream.Debounce(300*time.Millisecond).
			Distinct(func(term interface{}) interface{} {
				return term
			}).
			Subscribe(ctx,
				func(term interface{}) {
					searchTerm, ok := term.(string)
					if !ok || searchTerm == "" {
						return
					}

					var sims []models.Sim
					query := "%" + searchTerm + "%"
					if err := database.DB.Where(
						"iccid LIKE ? OR msisdn LIKE ? OR imsi LIKE ?",
						query, query, query,
					).Find(&sims).Error; err != nil {
						resultCh <- rxgo.Error(err)
						return
					}

					for _, sim := range sims {
						resultCh <- rxgo.Of(sim)
					}
				},
				func(err error) {
					resultCh <- rxgo.Error(err)
				},
				nil,
			)
	}()

	return NewStream(ctx, resultCh, r.config)
}
