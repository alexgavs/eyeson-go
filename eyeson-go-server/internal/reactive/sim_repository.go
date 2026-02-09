// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Repository

package reactive

import (
	"context"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"

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
