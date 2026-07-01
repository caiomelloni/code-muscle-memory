package scheduler

import (
	"testing"
	"time"
)

func TestSM2FailureIsDueSoon(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	got := NewSM2().Review(now, Progress{ExerciseID: "go-001"}, Again)

	if got.Lapses != 1 {
		t.Fatalf("Lapses = %d, want 1", got.Lapses)
	}
	if got.DueAt.Sub(now) != 10*time.Minute {
		t.Fatalf("DueAt = %s, want 10 minutes after now", got.DueAt)
	}
	if got.IntervalDays != 0 {
		t.Fatalf("IntervalDays = %d, want 0", got.IntervalDays)
	}
}

func TestSM2SuccessfulReviewsIncreaseInterval(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	s := NewSM2()

	first := s.Review(now, Progress{ExerciseID: "go-001"}, Good)
	if first.IntervalDays != 1 {
		t.Fatalf("first interval = %d, want 1", first.IntervalDays)
	}

	second := s.Review(now.AddDate(0, 0, 1), first, Good)
	if second.IntervalDays != 3 {
		t.Fatalf("second interval = %d, want 3", second.IntervalDays)
	}

	easy := s.Review(now, Progress{ExerciseID: "go-002"}, Easy)
	if easy.IntervalDays != 4 {
		t.Fatalf("easy first interval = %d, want 4", easy.IntervalDays)
	}
}
