package scheduler

import (
	"math"
	"math/rand/v2"
	"time"
)

type Rating string

const (
	Again Rating = "again"
	Hard  Rating = "hard"
	Good  Rating = "good"
	Easy  Rating = "easy"
)

type State string

const (
	StateLearning   State = "learning"
	StateReview     State = "review"
	StateRelearning State = "relearning"
)

const (
	minEase     = 1.3
	defaultEase = 2.5
)

type Review struct {
	ReviewedAt time.Time `json:"reviewed_at"`
	Rating     Rating    `json:"rating"`
	Passed     bool      `json:"passed"`
	Interval   int       `json:"interval_days"`
	EaseFactor float64   `json:"ease_factor"`
}

type Progress struct {
	ExerciseID     string    `json:"exercise_id"`
	State          State     `json:"state,omitempty"`
	LearningStep   int       `json:"learning_step,omitempty"`
	LapseInterval  int       `json:"lapse_interval_days,omitempty"`
	DueAt          time.Time `json:"due_at"`
	IntervalDays   int       `json:"interval_days"`
	EaseFactor     float64   `json:"ease_factor"`
	ReviewCount    int       `json:"review_count"`
	Lapses         int       `json:"lapses"`
	LastReviewedAt time.Time `json:"last_reviewed_at,omitempty"`
	History        []Review  `json:"history,omitempty"`
}

// CurrentState reports the card's phase, deriving it for progress records
// written before states were stored.
func (p Progress) CurrentState() State {
	if p.State != "" {
		return p.State
	}
	if p.IntervalDays >= 1 {
		return StateReview
	}
	return StateLearning
}

type Scheduler interface {
	Review(now time.Time, current Progress, rating Rating) Progress
	Preview(now time.Time, current Progress) map[Rating]time.Duration
}

// SM2 implements Anki's classic scheduler: cards move through intra-day
// learning steps, graduate to day-based review intervals grown by an ease
// factor, and drop into relearning steps when a review lapses.
type SM2 struct {
	LearningSteps       []time.Duration
	RelearningSteps     []time.Duration
	GraduatingInterval  int
	EasyInterval        int
	HardMultiplier      float64
	EasyBonus           float64
	MaxInterval         int
	LapseNewIntervalPct float64
	Fuzz                func(days int) int
}

func NewSM2() SM2 {
	return SM2{
		LearningSteps:       []time.Duration{10 * time.Minute},
		RelearningSteps:     []time.Duration{10 * time.Minute},
		GraduatingInterval:  1,
		EasyInterval:        4,
		HardMultiplier:      1.2,
		EasyBonus:           1.3,
		MaxInterval:         365,
		LapseNewIntervalPct: 0,
		Fuzz:                defaultFuzz,
	}
}

func (s SM2) Review(now time.Time, current Progress, rating Rating) Progress {
	if current.EaseFactor == 0 {
		current.EaseFactor = defaultEase
	}

	switch current.CurrentState() {
	case StateReview:
		current = s.reviewCard(now, current, rating)
	case StateRelearning:
		relearnInterval := max(1, int(math.Round(float64(current.LapseInterval)*s.LapseNewIntervalPct)))
		current = s.stepCard(now, current, rating, StateRelearning, s.RelearningSteps, relearnInterval, relearnInterval)
	default:
		current = s.stepCard(now, current, rating, StateLearning, s.LearningSteps, s.GraduatingInterval, s.EasyInterval)
	}

	current.ReviewCount++
	current.LastReviewedAt = now
	current.History = append(current.History, Review{
		ReviewedAt: now,
		Rating:     rating,
		Passed:     rating != Again,
		Interval:   current.IntervalDays,
		EaseFactor: current.EaseFactor,
	})
	return current
}

// Preview reports how far away the next review would land for each rating,
// with fuzz disabled so the numbers are stable enough to show the user.
func (s SM2) Preview(now time.Time, current Progress) map[Rating]time.Duration {
	s.Fuzz = func(days int) int { return days }
	previews := make(map[Rating]time.Duration, 4)
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		previews[rating] = s.Review(now, current, rating).DueAt.Sub(now)
	}
	return previews
}

// stepCard answers a card that is inside learning or relearning steps. Ease
// is left untouched: only review-phase answers move it.
func (s SM2) stepCard(now time.Time, p Progress, rating Rating, state State, steps []time.Duration, goodInterval, easyInterval int) Progress {
	p.State = state
	p.IntervalDays = 0
	switch rating {
	case Again:
		p.LearningStep = 0
		p.DueAt = now.Add(steps[0])
	case Hard:
		p.DueAt = now.Add(steps[min(p.LearningStep, len(steps)-1)])
	case Easy:
		return s.graduate(now, p, easyInterval)
	default:
		next := p.LearningStep + 1
		if p.ReviewCount == 0 {
			// A brand-new card enters the first step instead of skipping it.
			next = 0
		}
		if next >= len(steps) {
			return s.graduate(now, p, goodInterval)
		}
		p.LearningStep = next
		p.DueAt = now.Add(steps[next])
	}
	return p
}

func (s SM2) graduate(now time.Time, p Progress, intervalDays int) Progress {
	p.State = StateReview
	p.LearningStep = 0
	p.LapseInterval = 0
	p.IntervalDays = s.clampInterval(intervalDays)
	p.DueAt = now.AddDate(0, 0, p.IntervalDays)
	return p
}

func (s SM2) reviewCard(now time.Time, p Progress, rating Rating) Progress {
	p.State = StateReview
	if rating == Again {
		p.Lapses++
		p.EaseFactor = math.Max(minEase, p.EaseFactor-0.20)
		p.LapseInterval = p.IntervalDays
		p.State = StateRelearning
		p.LearningStep = 0
		p.IntervalDays = 0
		p.DueAt = now.Add(s.RelearningSteps[0])
		return p
	}

	delay := overdueDays(now, p.DueAt)
	interval := float64(p.IntervalDays)
	switch rating {
	case Hard:
		p.EaseFactor = math.Max(minEase, p.EaseFactor-0.15)
		interval *= s.HardMultiplier
	case Easy:
		interval = (interval + float64(delay)) * p.EaseFactor * s.EasyBonus
		p.EaseFactor += 0.15
	default:
		interval = (interval + float64(delay)/2) * p.EaseFactor
	}

	next := int(math.Ceil(interval))
	if next <= p.IntervalDays {
		next = p.IntervalDays + 1
	}
	next = s.clampInterval(s.fuzz(s.clampInterval(next)))
	p.IntervalDays = next
	p.DueAt = now.AddDate(0, 0, next)
	return p
}

func (s SM2) clampInterval(days int) int {
	days = max(1, days)
	if s.MaxInterval > 0 {
		days = min(days, s.MaxInterval)
	}
	return days
}

func (s SM2) fuzz(days int) int {
	if s.Fuzz == nil {
		return days
	}
	return s.Fuzz(days)
}

// defaultFuzz spreads intervals slightly so cards reviewed together do not
// stay due on the same day forever.
func defaultFuzz(days int) int {
	if days < 3 {
		return days
	}
	spread := 1
	if days > 7 {
		spread = max(1, int(float64(days)*0.05))
	}
	return days + rand.IntN(2*spread+1) - spread
}

func overdueDays(now time.Time, due time.Time) int {
	if due.IsZero() || !now.After(due) {
		return 0
	}
	return int(now.Sub(due).Hours() / 24)
}
