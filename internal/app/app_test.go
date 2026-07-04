package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"code-muscle-memory/internal/exercise"
	"code-muscle-memory/internal/scheduler"
	"code-muscle-memory/internal/storage"
)

func TestChooseNextPrefersDueReviewsOverNew(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	exercises := []exercise.Exercise{
		{ID: "go-002", Difficulty: 2},
		{ID: "go-001", Difficulty: 1},
	}
	progress := storage.ProgressFile{
		Items: map[string]scheduler.Progress{
			"go-002": {ExerciseID: "go-002", State: scheduler.StateReview, IntervalDays: 3, DueAt: now.Add(-time.Hour)},
		},
	}

	got, ok := chooseNext(exercises, progress, now)
	if !ok {
		t.Fatal("chooseNext returned false")
	}
	if got.ID != "go-002" {
		t.Fatalf("chosen = %q, want the due review go-002 before the new go-001", got.ID)
	}
}

func TestChooseNextPrefersDueLearningOverReviews(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	exercises := []exercise.Exercise{
		{ID: "go-001", Difficulty: 1},
		{ID: "go-002", Difficulty: 2},
	}
	progress := storage.ProgressFile{
		Items: map[string]scheduler.Progress{
			"go-001": {ExerciseID: "go-001", State: scheduler.StateReview, IntervalDays: 3, DueAt: now.Add(-2 * time.Hour)},
			"go-002": {ExerciseID: "go-002", State: scheduler.StateLearning, DueAt: now.Add(-time.Minute)},
		},
	}

	got, ok := chooseNext(exercises, progress, now)
	if !ok {
		t.Fatal("chooseNext returned false")
	}
	if got.ID != "go-002" {
		t.Fatalf("chosen = %q, want the due learning card go-002 before the due review go-001", got.ID)
	}
}

func TestChooseNextOrdersNewExercisesByDifficulty(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	exercises := []exercise.Exercise{
		{ID: "go-002", Difficulty: 2},
		{ID: "go-001", Difficulty: 1},
	}

	got, ok := chooseNext(exercises, storage.ProgressFile{Items: map[string]scheduler.Progress{}}, now)
	if !ok {
		t.Fatal("chooseNext returned false")
	}
	if got.ID != "go-001" {
		t.Fatalf("chosen = %q, want the easiest new exercise go-001", got.ID)
	}
}

func TestFindExercise(t *testing.T) {
	exercises := []exercise.Exercise{
		{ID: "go-001", Title: "A"},
		{ID: "go-002", Title: "B"},
	}

	got, ok := findExercise(exercises, "go-002")
	if !ok {
		t.Fatal("findExercise returned false for an existing id")
	}
	if got.Title != "B" {
		t.Fatalf("Title = %q, want %q", got.Title, "B")
	}

	if _, ok := findExercise(exercises, "missing"); ok {
		t.Fatal("findExercise returned true for a missing id")
	}
}

func TestReviewStatus(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	if got, want := reviewStatus(scheduler.Progress{}, false, now), "new"; got != want {
		t.Fatalf("reviewStatus() = %q, want %q", got, want)
	}
	reviewCard := scheduler.Progress{State: scheduler.StateReview, IntervalDays: 1}
	reviewCard.DueAt = now.Add(24 * time.Hour)
	if got, want := reviewStatus(reviewCard, true, now), "due 2026-07-02"; got != want {
		t.Fatalf("reviewStatus() = %q, want %q", got, want)
	}
	reviewCard.DueAt = now.Add(-time.Hour)
	if got, want := reviewStatus(reviewCard, true, now), "due now"; got != want {
		t.Fatalf("reviewStatus() = %q, want %q", got, want)
	}
	learningCard := scheduler.Progress{State: scheduler.StateLearning, DueAt: now.Add(10 * time.Minute)}
	if got, want := reviewStatus(learningCard, true, now), "learning (due 12:10)"; got != want {
		t.Fatalf("reviewStatus() = %q, want %q", got, want)
	}
	relearningCard := scheduler.Progress{State: scheduler.StateRelearning, DueAt: now.Add(-time.Minute)}
	if got, want := reviewStatus(relearningCard, true, now), "relearning (due now)"; got != want {
		t.Fatalf("reviewStatus() = %q, want %q", got, want)
	}
}

func TestPromptRatingShowsPreviewsAndParsesAgain(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	var out strings.Builder
	a := New(Config{
		Stdout: &out,
		Stdin:  strings.NewReader("a\n"),
		Now:    func() time.Time { return now },
	})
	item := scheduler.Progress{ExerciseID: "go-001", State: scheduler.StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	if got := a.promptRating(now, item); got != scheduler.Again {
		t.Fatalf("promptRating() = %q, want again", got)
	}
	prompt := out.String()
	for _, want := range []string{"[a]gain 10m", "[h]ard 12d", "[g]ood 25d", "[e]asy 33d"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestPromptRatingWarnsAndRetriesOnInvalidInput(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	var out strings.Builder
	a := New(Config{
		Stdout: &out,
		Stdin:  strings.NewReader("banana\nh\n"),
		Now:    func() time.Time { return now },
	})
	item := scheduler.Progress{ExerciseID: "go-001", State: scheduler.StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	if got := a.promptRating(now, item); got != scheduler.Hard {
		t.Fatalf("promptRating() = %q, want hard after retry", got)
	}
	prompt := out.String()
	if !strings.Contains(prompt, `Unknown rating "banana"`) {
		t.Fatalf("prompt missing warning about invalid input:\n%s", prompt)
	}
	if strings.Count(prompt, "Rate this review") != 2 {
		t.Fatalf("prompt should be shown twice (initial + retry):\n%s", prompt)
	}
}

func TestPromptRatingRejectsEmptyInput(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	var out strings.Builder
	a := New(Config{
		Stdout: &out,
		Stdin:  strings.NewReader("\ne\n"),
		Now:    func() time.Time { return now },
	})
	item := scheduler.Progress{ExerciseID: "go-001", State: scheduler.StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	if got := a.promptRating(now, item); got != scheduler.Easy {
		t.Fatalf("promptRating() = %q, want easy after empty input is rejected", got)
	}
	prompt := out.String()
	if !strings.Contains(prompt, "A rating is required") {
		t.Fatalf("prompt missing warning about empty input:\n%s", prompt)
	}
	if strings.Count(prompt, "Rate this review") != 2 {
		t.Fatalf("prompt should be shown twice (initial + retry):\n%s", prompt)
	}
}

func TestPromptRatingDefaultsToGoodWhenInputRunsOut(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	a := New(Config{
		Stdout: &strings.Builder{},
		Stdin:  strings.NewReader(""),
		Now:    func() time.Time { return now },
	})
	item := scheduler.Progress{ExerciseID: "go-001", State: scheduler.StateReview, IntervalDays: 10, EaseFactor: 2.5, DueAt: now}

	if got := a.promptRating(now, item); got != scheduler.Good {
		t.Fatalf("promptRating() = %q, want good on EOF", got)
	}
}

func TestWriteStarterPrependsInstructions(t *testing.T) {
	ex := exercise.Exercise{
		Title:       "Return a greeting",
		Description: "Implement Hello.",
		Objective:   "Practice functions.",
		StarterCode: "package exercise\n\nfunc Hello(name string) string { return \"\" }\n",
	}

	path, cleanup, err := writeStarter(ex, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		"// Return a greeting",
		"// What to do: Implement Hello.",
		"// Objective: Practice functions.",
		"// Implement: a function called \"Hello\", receives a parameter called \"name\" of type string, and returns a string",
		"package exercise",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("starter file missing %q:\n%s", want, got)
		}
	}
	if !strings.HasPrefix(got, "// Return a greeting") {
		t.Fatalf("starter file should begin with instruction comments:\n%s", got)
	}
	if strings.Contains(got, "func Hello") || strings.Contains(got, "return \"\"") || strings.Contains(got, "{ return") {
		t.Fatalf("starter file should not include starter code:\n%s", got)
	}
}

func TestWriteStarterRestoresPreviousAttempt(t *testing.T) {
	ex := exercise.Exercise{
		Title:       "Return a greeting",
		Description: "Implement Hello.",
		StarterCode: "package exercise\n\nfunc Hello(name string) string { return \"\" }\n",
	}
	previousAttempt := "package exercise\n\nfunc Hello(name string) string {\n\treturn \"partial attempt\"\n}\n"

	path, cleanup, err := writeStarter(ex, previousAttempt)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != previousAttempt {
		t.Fatalf("starter file = %q, want previous attempt restored verbatim %q", got, previousAttempt)
	}
}

func TestStarterInstructionsIncludeRequiredTypesAndMethods(t *testing.T) {
	ex := exercise.Exercise{
		Title:       "Calculate rectangle area",
		Description: "Define an Area method.",
		StarterCode: "package exercise\n\ntype Rectangle struct {\n\tWidth float64\n\tHeight float64\n}\n\nfunc (r Rectangle) Area() float64 {\n\treturn 0\n}\n",
	}

	got := starterWithInstructions(ex)
	for _, want := range []string{
		"// Implement: a struct type called \"Rectangle\" with a field called \"Width\" of type float64 and a field called \"Height\" of type float64",
		"// Implement: a method called \"Area\" on Rectangle, receives no parameters, and returns a float64",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("starter instructions missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "func ") || strings.Contains(got, "return 0") {
		t.Fatalf("starter file should not include method implementation:\n%s", got)
	}
}

func TestStarterInstructionsForTestWritingUseGivenPrefix(t *testing.T) {
	ex := exercise.Exercise{
		Title:       "Test a doubling function",
		Description: "Write a test for Double.",
		Objective:   "Practice writing a basic Go test.",
		Kind:        exercise.KindTestWriting,
		SubjectCode: "package exercise\n\nfunc Double(n int) int {\n\treturn n * 2\n}\n",
	}

	got := starterWithInstructions(ex)
	for _, want := range []string{
		"// Given: a function called \"Double\", receives a parameter called \"n\" of type int, and returns an int",
		"// Write: one or more functions whose names start with \"Test\", each receiving a parameter called \"t\" of type pointer to testing.T and returning nothing",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("starter instructions missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Implement:") {
		t.Fatalf("starter instructions should not use the Implement: prefix for test-writing exercises:\n%s", got)
	}
	if strings.Contains(got, "func Double") || strings.Contains(got, "return n * 2") {
		t.Fatalf("starter file should not include subject code:\n%s", got)
	}
}

func TestStarterInstructionsIncludeRequiredTestFeatures(t *testing.T) {
	ex := exercise.Exercise{
		Title:                "Test FizzBuzz with named subtests",
		Description:          "Write a test for FizzBuzz.",
		Kind:                 exercise.KindTestWriting,
		SubjectCode:          "package exercise\n\nfunc Double(n int) int {\n\treturn n * 2\n}\n",
		RequiredTestFeatures: []string{exercise.FeatureTable, exercise.FeatureSubtests},
	}

	got := starterWithInstructions(ex)
	for _, want := range []string{
		"// Use: a table of test cases driven by a loop",
		"// Use: named subtests, giving each case its own name",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("starter instructions missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "t.Run") {
		t.Fatalf("starter instructions should not leak Go syntax:\n%s", got)
	}
}

func TestChooseNextFallsBackToEarliestFutureDue(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	exercises := []exercise.Exercise{
		{ID: "go-001", Difficulty: 1},
		{ID: "go-002", Difficulty: 2},
	}
	progress := storage.ProgressFile{
		Items: map[string]scheduler.Progress{
			"go-001": {ExerciseID: "go-001", DueAt: now.Add(48 * time.Hour)},
			"go-002": {ExerciseID: "go-002", DueAt: now.Add(24 * time.Hour)},
		},
	}

	got, ok := chooseNext(exercises, progress, now)
	if !ok {
		t.Fatal("chooseNext returned false")
	}
	if got.ID != "go-002" {
		t.Fatalf("chosen = %q, want go-002", got.ID)
	}
}

func TestChooseNextKeepsCurrentExerciseUntilSolved(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	exercises := []exercise.Exercise{
		{ID: "go-001", Difficulty: 1},
		{ID: "go-002", Difficulty: 2},
	}
	progress := storage.ProgressFile{
		CurrentExerciseID: "go-001",
		Items: map[string]scheduler.Progress{
			"go-001": {ExerciseID: "go-001", DueAt: now.Add(48 * time.Hour)},
			"go-002": {ExerciseID: "go-002", DueAt: now},
		},
	}

	got, ok := chooseNext(exercises, progress, now)
	if !ok {
		t.Fatal("chooseNext returned false")
	}
	if got.ID != "go-001" {
		t.Fatalf("chosen = %q, want go-001", got.ID)
	}
}
