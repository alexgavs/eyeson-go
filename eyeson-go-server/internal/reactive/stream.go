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

// FlatMap applies a function that returns an observable for each item
func (s *Stream) FlatMap(transform func(interface{}) rxgo.Observable) *Stream {
	s.observable = s.observable.FlatMap(func(i rxgo.Item) rxgo.Observable {
		return transform(i.V)
	})
	return s
}

// Buffer collects items into batches
func (s *Stream) Buffer(count int, timespan time.Duration) *Stream {
	s.observable = s.observable.BufferWithTimeOrCount(
		rxgo.WithDuration(timespan),
		count,
	)
	return s
}

// Debounce emits an item only if a particular timespan has passed without another emission
func (s *Stream) Debounce(timespan time.Duration) *Stream {
	s.observable = s.observable.Debounce(rxgo.WithDuration(timespan))
	return s
}

// Distinct filters out duplicate items
func (s *Stream) Distinct(keySelector func(interface{}) interface{}) *Stream {
	s.observable = s.observable.Distinct(func(_ context.Context, i interface{}) (interface{}, error) {
		return keySelector(i), nil
	})
	return s
}

// Retry retries failed operations
func (s *Stream) Retry(attempts int) *Stream {
	s.observable = s.observable.Retry(attempts, func(err error) bool {
		return true // Retry all errors
	})
	return s
}

// Timeout adds a timeout to the stream (removed as not available in rxgo v2.5.0)
// Use context.WithTimeout instead
// func (s *Stream) Timeout(duration time.Duration) *Stream {
// 	return s
// }

// CatchError handles errors with a fallback function
func (s *Stream) CatchError(fallback func(error) interface{}) *Stream {
	s.observable = s.observable.OnErrorReturn(func(err error) interface{} {
		return fallback(err)
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
