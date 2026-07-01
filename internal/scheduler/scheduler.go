package scheduler

import (
	"math"
	"time"
)

type Rating string

const (
	Again Rating = "again"
	Hard  Rating = "hard"
	Good  Rating = "good"
	Easy  Rating = "easy"
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
	DueAt          time.Time `json:"due_at"`
	IntervalDays   int       `json:"interval_days"`
	EaseFactor     float64   `json:"ease_factor"`
	ReviewCount    int       `json:"review_count"`
	Lapses         int       `json:"lapses"`
	LastReviewedAt time.Time `json:"last_reviewed_at,omitempty"`
	History        []Review  `json:"history,omitempty"`
}

type Scheduler interface {
	Review(now time.Time, current Progress, rating Rating) Progress
}

type SM2 struct{}

func NewSM2() SM2 {
	return SM2{}
}

func (SM2) Review(now time.Time, current Progress, rating Rating) Progress {
	if current.EaseFactor == 0 {
		current.EaseFactor = 2.5
	}

	passed := rating != Again
	switch rating {
	case Again:
		current.Lapses++
		current.IntervalDays = 0
		current.EaseFactor = math.Max(1.3, current.EaseFactor-0.2)
		current.DueAt = now.Add(10 * time.Minute)
	case Hard:
		current.IntervalDays = maxInt(1, current.IntervalDays)
		current.EaseFactor = math.Max(1.3, current.EaseFactor-0.15)
		current.DueAt = now.AddDate(0, 0, current.IntervalDays)
	case Easy:
		if current.IntervalDays == 0 {
			current.IntervalDays = 4
		} else {
			current.IntervalDays = int(math.Ceil(float64(current.IntervalDays) * current.EaseFactor * 1.3))
		}
		current.EaseFactor += 0.15
		current.DueAt = now.AddDate(0, 0, current.IntervalDays)
	default:
		if current.IntervalDays == 0 {
			current.IntervalDays = 1
		} else if current.IntervalDays == 1 {
			current.IntervalDays = 3
		} else {
			current.IntervalDays = int(math.Ceil(float64(current.IntervalDays) * current.EaseFactor))
		}
		current.DueAt = now.AddDate(0, 0, current.IntervalDays)
	}

	current.ReviewCount++
	current.LastReviewedAt = now
	current.History = append(current.History, Review{
		ReviewedAt: now,
		Rating:     rating,
		Passed:     passed,
		Interval:   current.IntervalDays,
		EaseFactor: current.EaseFactor,
	})
	return current
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
