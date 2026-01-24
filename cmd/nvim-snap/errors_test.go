package main

import (
	"errors"
	"testing"
)

func TestExitCode(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := exitCode(nil); got != 0 {
			t.Fatalf("exitCode(nil) = %d, want 0", got)
		}
	})
	t.Run("generic", func(t *testing.T) {
		if got := exitCode(errors.New("boom")); got != 1 {
			t.Fatalf("exitCode(generic) = %d, want 1", got)
		}
	})
	t.Run("exit-error", func(t *testing.T) {
		err := ExitError{Code: 2, Err: errors.New("nope")}
		if got := exitCode(err); got != 2 {
			t.Fatalf("exitCode(exit) = %d, want 2", got)
		}
	})
	t.Run("exit-zero", func(t *testing.T) {
		err := ExitError{Code: 0, Err: errors.New("nope")}
		if got := exitCode(err); got != 1 {
			t.Fatalf("exitCode(exit zero) = %d, want 1", got)
		}
	})
}

func TestIsSilent(t *testing.T) {
	if isSilent(nil) {
		t.Fatalf("isSilent(nil) = true, want false")
	}
	if isSilent(errors.New("boom")) {
		t.Fatalf("isSilent(generic) = true, want false")
	}
	if !isSilent(ExitError{Code: 2, Err: errors.New("nope"), Silent: true}) {
		t.Fatalf("isSilent(exit) = false, want true")
	}
	if isSilent(ExitError{Code: 2, Err: errors.New("nope"), Silent: false}) {
		t.Fatalf("isSilent(exit) = true, want false")
	}
}
