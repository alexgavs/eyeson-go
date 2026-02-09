// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System - Reactive Core

package reactive

import (
	"context"
	"time"

	"github.com/reactivex/rxgo/v2"
)

// StreamConfig holds configuration for reactive streams
type StreamConfig struct {
	BufferSize    int
	WorkerCount   int
	RetryAttempts int
	RetryDelay    time.Duration
}

// DefaultStreamConfig returns default configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		BufferSize:    100,
		WorkerCount:   4,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
}

// Stream represents a reactive data stream
type Stream struct {
	observable rxgo.Observable
	config     StreamConfig
}

// NewStream creates a new reactive stream from a channel
func NewStream(ctx context.Context, source <-chan rxgo.Item, config StreamConfig) *Stream {
	obs := rxgo.FromChannel(source, rxgo.WithContext(ctx))
	return &Stream{
		observable: obs,
		config:     config,
	}
}

// Map applies a transformation function to each item
func (s *Stream) Map(transform func(interface{}) interface{}) *Stream {
	s.observable = s.observable.Map(func(_ context.Context, i interface{}) (interface{}, error) {
		return transform(i), nil
	})
	return s
}

// Filter applies a predicate to filter items
func (s *Stream) Filter(predicate func(interface{}) bool) *Stream {
	s.observable = s.observable.Filter(func(i interface{}) bool {
		return predicate(i)
	})
	return s
}

// Subscribe subscribes to the stream with handlers
func (s *Stream) Subscribe(
	ctx context.Context,
	onNext func(interface{}),
	onError func(error),
	onComplete func(),
) {
	s.observable.ForEach(
		func(v interface{}) {
			onNext(v)
		},
		func(err error) {
			if onError != nil {
				onError(err)
			}
		},
		func() {
			if onComplete != nil {
				onComplete()
			}
		},
	)
}

// ToChannel converts the stream back to a channel
func (s *Stream) ToChannel() <-chan rxgo.Item {
	return s.observable.Observe()
}

// Observable returns the underlying rxgo.Observable
func (s *Stream) Observable() rxgo.Observable {
	return s.observable
}
