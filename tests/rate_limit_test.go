package tests

import (
	"sync"
	"testing"

	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/store"
)

func TestRateLimiter_Concurrency(t *testing.T) {
	s := store.NewInMemoryStore()
	userID := "test-user"
	var wg sync.WaitGroup
	numReqs := 10

	for i := 0; i < numReqs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.AllowRequest(userID)
		}()
	}
	wg.Wait()

	stats := s.GetStats()[userID]
	if stats.AcceptedCurrentWindow != 5 {
		t.Errorf("Expected 5 accepted requests, got %d", stats.AcceptedCurrentWindow)
	}
	if stats.RejectedCumulative != 5 {
		t.Errorf("Expected 5 rejected requests, got %d", stats.RejectedCumulative)
	}
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	s := store.NewInMemoryStore()
	userID := "window-user"

	for i := 0; i < 5; i++ {
		if !s.AllowRequest(userID) {
			t.Fatal("Should allow first 5")
		}
	}
	if s.AllowRequest(userID) {
		t.Fatal("Should reject 6th")
	}
}
