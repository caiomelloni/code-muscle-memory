package scheduler

import (
	"testing"
	"time"
)

func newTestSM2() SM2 {
	s := NewSM2()
	s.Fuzz = func(days int) int { return days }
	return s
}

func TestNewCardWalksLearningStepsThenGraduates(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSM2()

	first := s.Review(now, Progress{ExerciseID: "go-001"}, Good)
	if first.CurrentState() != StateLearning {
		t.Fatalf("state = %q, want learning", first.CurrentState())
	}
	if got := first.DueAt.Sub(now); got != 10*time.Minute {
		t.Fatalf("first due in %s, want 10m", got)
	}
	if first.IntervalDays != 0 {
		t.Fatalf("IntervalDays = %d, want 0 during learning", first.IntervalDays)
	}

	later := now.Add(10 * time.Minute)
	second := s.Review(later, first, Good)
	if second.CurrentState() != StateReview {
		t.Fatalf("state = %q, want review after graduating", second.CurrentState())
	}
	if second.IntervalDays != 1 {
		t.Fatalf("graduating interval = %d, want 1", second.IntervalDays)
	}

	nextDay := later.AddDate(0, 0, 1)
	third := s.Review(nextDay, second, Good)
	if third.IntervalDays != 3 {
		t.Fatalf("first review interval = %d, want 3 (1 x 2.5 rounded up)", third.IntervalDays)
	}
	if third.EaseFactor != 2.5 {
		t.Fatalf("EaseFactor = %v, want 2.5 unchanged by Good", third.EaseFactor)
	}
}

func TestNewCardEasyGraduatesImmediately(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	got := newTestSM2().Review(now, Progress{ExerciseID: "go-001"}, Easy)

	if got.CurrentState() != StateReview {
		t.Fatalf("state = %q, want review", got.CurrentState())
	}
	if got.IntervalDays != 4 {
		t.Fatalf("IntervalDays = %d, want easy interval 4", got.IntervalDays)
	}
	if got.EaseFactor != 2.5 {
		t.Fatalf("EaseFactor = %v, want 2.5 untouched during learning", got.EaseFactor)
	}
}

func TestLearningAgainAndHardRepeatSteps(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSM2()
	card := s.Review(now, Progress{ExerciseID: "go-001"}, Good)

	again := s.Review(now.Add(10*time.Minute), card, Again)
	if again.CurrentState() != StateLearning {
		t.Fatalf("state = %q, want learning", again.CurrentState())
	}
	if again.Lapses != 0 {
		t.Fatalf("Lapses = %d, want 0: learning failures are not lapses", again.Lapses)
	}
	if got := again.DueAt.Sub(now.Add(10 * time.Minute)); got != 10*time.Minute {
		t.Fatalf("again due in %s, want 10m", got)
	}

	hard := s.Review(now.Add(10*time.Minute), card, Hard)
	if hard.CurrentState() != StateLearning {
		t.Fatalf("state = %q, want learning", hard.CurrentState())
	}
	if got := hard.DueAt.Sub(now.Add(10 * time.Minute)); got != 10*time.Minute {
		t.Fatalf("hard due in %s, want current step repeated (10m)", got)
	}
}

func TestReviewHardGrowsIntervalAndReducesEase(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	card := Progress{ExerciseID: "go-001", State: StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	got := newTestSM2().Review(now, card, Hard)
	if got.IntervalDays != 12 {
		t.Fatalf("IntervalDays = %d, want 12 (10 x 1.2)", got.IntervalDays)
	}
	if got.EaseFactor != 2.35 {
		t.Fatalf("EaseFactor = %v, want 2.35", got.EaseFactor)
	}
}

func TestReviewGoodGivesOverdueCredit(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	card := Progress{ExerciseID: "go-001", State: StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now.AddDate(0, 0, -4)}

	got := newTestSM2().Review(now, card, Good)
	if got.IntervalDays != 30 {
		t.Fatalf("IntervalDays = %d, want 30 ((10 + 4/2) x 2.5)", got.IntervalDays)
	}
}

func TestReviewEasyAppliesBonusAndRaisesEase(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	card := Progress{ExerciseID: "go-001", State: StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	got := newTestSM2().Review(now, card, Easy)
	if got.IntervalDays != 33 {
		t.Fatalf("IntervalDays = %d, want 33 (10 x 2.5 x 1.3 rounded up)", got.IntervalDays)
	}
	if got.EaseFactor != 2.65 {
		t.Fatalf("EaseFactor = %v, want 2.65", got.EaseFactor)
	}
}

func TestReviewAgainLapsesIntoRelearning(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSM2()
	card := Progress{ExerciseID: "go-001", State: StateReview, IntervalDays: 20, EaseFactor: 2.5, DueAt: now}

	lapsed := s.Review(now, card, Again)
	if lapsed.CurrentState() != StateRelearning {
		t.Fatalf("state = %q, want relearning", lapsed.CurrentState())
	}
	if lapsed.Lapses != 1 {
		t.Fatalf("Lapses = %d, want 1", lapsed.Lapses)
	}
	if lapsed.EaseFactor != 2.3 {
		t.Fatalf("EaseFactor = %v, want 2.3", lapsed.EaseFactor)
	}
	if lapsed.LapseInterval != 20 {
		t.Fatalf("LapseInterval = %d, want 20", lapsed.LapseInterval)
	}
	if got := lapsed.DueAt.Sub(now); got != 10*time.Minute {
		t.Fatalf("due in %s, want 10m", got)
	}

	later := now.Add(10 * time.Minute)
	recovered := s.Review(later, lapsed, Good)
	if recovered.CurrentState() != StateReview {
		t.Fatalf("state = %q, want review after relearning", recovered.CurrentState())
	}
	if recovered.IntervalDays != 1 {
		t.Fatalf("post-relearn interval = %d, want 1 (lapse pct 0)", recovered.IntervalDays)
	}
}

func TestReviewIntervalIsCappedAtMaxInterval(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	card := Progress{ExerciseID: "go-001", State: StateReview, IntervalDays: 300, EaseFactor: 2.5, DueAt: now}

	got := newTestSM2().Review(now, card, Good)
	if got.IntervalDays != 365 {
		t.Fatalf("IntervalDays = %d, want capped at 365", got.IntervalDays)
	}
}

func TestReviewAppliesFuzz(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	s := NewSM2()
	s.Fuzz = func(days int) int { return days + 2 }
	card := Progress{ExerciseID: "go-001", State: StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	got := s.Review(now, card, Good)
	if got.IntervalDays != 27 {
		t.Fatalf("IntervalDays = %d, want 27 (25 + fuzz 2)", got.IntervalDays)
	}
}

func TestLegacyProgressWithoutStateIsTreatedAsReview(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	card := Progress{ExerciseID: "go-001", IntervalDays: 6, EaseFactor: 2.5, DueAt: now, ReviewCount: 3}

	got := newTestSM2().Review(now, card, Good)
	if got.CurrentState() != StateReview {
		t.Fatalf("state = %q, want review derived from interval", got.CurrentState())
	}
	if got.IntervalDays != 15 {
		t.Fatalf("IntervalDays = %d, want 15 (6 x 2.5)", got.IntervalDays)
	}
}

func TestPreviewCoversAllRatingsWithoutFuzz(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	s := NewSM2()
	s.Fuzz = func(days int) int { return days + 100 }
	card := Progress{ExerciseID: "go-001", State: StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	previews := s.Preview(now, card)
	want := map[Rating]time.Duration{
		Again: 10 * time.Minute,
		Hard:  12 * 24 * time.Hour,
		Good:  25 * 24 * time.Hour,
		Easy:  33 * 24 * time.Hour,
	}
	for rating, wantDelta := range want {
		if previews[rating] != wantDelta {
			t.Fatalf("preview[%s] = %s, want %s", rating, previews[rating], wantDelta)
		}
	}
}

func TestReviewRecordsHistory(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	got := newTestSM2().Review(now, Progress{ExerciseID: "go-001"}, Good)

	if got.ReviewCount != 1 {
		t.Fatalf("ReviewCount = %d, want 1", got.ReviewCount)
	}
	if len(got.History) != 1 {
		t.Fatalf("History length = %d, want 1", len(got.History))
	}
	if !got.History[0].Passed {
		t.Fatal("History[0].Passed = false, want true for Good")
	}
	if !got.LastReviewedAt.Equal(now) {
		t.Fatalf("LastReviewedAt = %s, want %s", got.LastReviewedAt, now)
	}
}

func TestDefaultFuzzStaysWithinBounds(t *testing.T) {
	for _, days := range []int{1, 2, 3, 7, 30, 365} {
		for range 50 {
			got := defaultFuzz(days)
			spread := 1
			if days > 7 {
				spread = max(1, int(float64(days)*0.05))
			}
			if days < 3 {
				spread = 0
			}
			if got < days-spread || got > days+spread {
				t.Fatalf("defaultFuzz(%d) = %d, want within +/- %d", days, got, spread)
			}
		}
	}
}
