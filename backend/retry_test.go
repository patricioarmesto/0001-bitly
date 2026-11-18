package main

import (
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsAfterTransientFailure(t *testing.T) {
	var delays []time.Duration
	attempts := 0
	err := retry(func(d time.Duration) error {
		delays = append(delays, d)
		return nil
	}, func() error {
		attempts++
		if attempts == 1 {
			return errors.New("fallo transitorio")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("retry devolvió error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("intentos = %d, esperaba 2", attempts)
	}
	if len(delays) != 1 || delays[0] != 200*time.Millisecond {
		t.Errorf("esperas = %v, esperaba [200ms]", delays)
	}
}

func TestRetryReturnsLastErrorAfterAllAttempts(t *testing.T) {
	sentinel := errors.New("siempre falla")
	var delays []time.Duration
	attempts := 0
	err := retry(func(d time.Duration) error {
		delays = append(delays, d)
		return nil
	}, func() error {
		attempts++
		return sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, esperaba %v", err, sentinel)
	}
	if attempts != 3 {
		t.Errorf("intentos = %d, esperaba 3", attempts)
	}
	want := []time.Duration{200 * time.Millisecond, 400 * time.Millisecond}
	if len(delays) != len(want) || delays[0] != want[0] || delays[1] != want[1] {
		t.Errorf("esperas = %v, esperaba %v", delays, want)
	}
}

func TestRetryStopsWhenWaitFails(t *testing.T) {
	stop := errors.New("contexto cancelado")
	attempts := 0
	err := retry(func(time.Duration) error { return stop }, func() error {
		attempts++
		return errors.New("fallo")
	})

	if !errors.Is(err, stop) {
		t.Errorf("err = %v, esperaba %v", err, stop)
	}
	if attempts != 1 {
		t.Errorf("intentos = %d, esperaba 1", attempts)
	}
}
