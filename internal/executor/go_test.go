package executor

import (
	"context"
	"strings"
	"testing"

	"code-muscle-memory/internal/exercise"
)

func TestGoExecutorSuccess(t *testing.T) {
	ex := addExercise()
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, "package exercise\n\nfunc Add(a, b int) int { return a + b }\n")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, Success, result.Output)
	}
}

func TestGoExecutorTestFailure(t *testing.T) {
	ex := addExercise()
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, "package exercise\n\nfunc Add(a, b int) int { return 0 }\n")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != TestFailure {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, TestFailure, result.Output)
	}
}

func TestGoExecutorCompileError(t *testing.T) {
	ex := addExercise()
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, "package exercise\n\nfunc Add(a, b int) int {")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CompileError {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, CompileError, result.Output)
	}
}

func TestGoExecutorTestWritingSuccess(t *testing.T) {
	ex := signExercise()
	testCode := `package exercise

import "testing"

func TestSign(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{n: 5, want: "positive"},
		{n: -5, want: "negative"},
		{n: 0, want: "zero"},
	}
	for _, tt := range tests {
		if got := Sign(tt.n); got != tt.want {
			t.Fatalf("Sign(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
`
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, testCode)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, Success, result.Output)
	}
}

func TestGoExecutorTestWritingMutantEscaped(t *testing.T) {
	ex := signExercise()
	testCode := `package exercise

import "testing"

func TestSign(t *testing.T) {
	if got := Sign(5); got != "positive" {
		t.Fatalf("Sign(5) = %q, want %q", got, "positive")
	}
	if got := Sign(-5); got != "negative" {
		t.Fatalf("Sign(-5) = %q, want %q", got, "negative")
	}
}
`
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, testCode)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != MutantEscaped {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, MutantEscaped, result.Output)
	}
	if !strings.Contains(result.Output, "misreports zero as positive") {
		t.Fatalf("Output = %q, want it to contain the mutant hint", result.Output)
	}
	if strings.Contains(result.Output, "func Sign") {
		t.Fatalf("Output leaked mutant source: %q", result.Output)
	}
}

func TestGoExecutorTestWritingCompileError(t *testing.T) {
	ex := signExercise()
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, "package exercise\n\nfunc TestSign(t *testing.T) {")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CompileError {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, CompileError, result.Output)
	}
}

const flatSignTest = `package exercise

import "testing"

func TestSign(t *testing.T) {
	if got := Sign(5); got != "positive" {
		t.Fatalf("Sign(5) = %q, want %q", got, "positive")
	}
	if got := Sign(-5); got != "negative" {
		t.Fatalf("Sign(-5) = %q, want %q", got, "negative")
	}
	if got := Sign(0); got != "zero" {
		t.Fatalf("Sign(0) = %q, want %q", got, "zero")
	}
}
`

func TestGoExecutorTestWritingMissingSubtests(t *testing.T) {
	ex := signExercise()
	ex.RequiredTestFeatures = []string{exercise.FeatureSubtests}
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, flatSignTest)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != MissingFeature {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, MissingFeature, result.Output)
	}
	if !strings.Contains(result.Output, "subtest") {
		t.Fatalf("Output = %q, want it to mention subtests", result.Output)
	}
}

func TestGoExecutorTestWritingMissingTable(t *testing.T) {
	ex := signExercise()
	ex.RequiredTestFeatures = []string{exercise.FeatureTable}
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, flatSignTest)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != MissingFeature {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, MissingFeature, result.Output)
	}
}

func TestGoExecutorTestWritingFeaturesSatisfied(t *testing.T) {
	ex := signExercise()
	ex.RequiredTestFeatures = []string{exercise.FeatureTable, exercise.FeatureSubtests}
	testCode := `package exercise

import "testing"

func TestSign(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "positive", n: 5, want: "positive"},
		{name: "negative", n: -5, want: "negative"},
		{name: "zero", n: 0, want: "zero"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sign(tt.n); got != tt.want {
				t.Fatalf("Sign(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
`
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, testCode)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, Success, result.Output)
	}
}

func TestGoExecutorTestWritingFeatureCheckedAfterCorrectness(t *testing.T) {
	ex := signExercise()
	ex.RequiredTestFeatures = []string{exercise.FeatureSubtests}
	testCode := `package exercise

import "testing"

func TestSign(t *testing.T) {
	if got := Sign(5); got != "negative" {
		t.Fatalf("Sign(5) = %q, want %q", got, "negative")
	}
}
`
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, testCode)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != TestFailure {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, TestFailure, result.Output)
	}
}

func TestGoExecutorTestWritingFailsOnCorrectSubject(t *testing.T) {
	ex := signExercise()
	testCode := `package exercise

import "testing"

func TestSign(t *testing.T) {
	if got := Sign(5); got != "negative" {
		t.Fatalf("Sign(5) = %q, want %q", got, "negative")
	}
}
`
	result, err := NewGoExecutor().Evaluate(context.Background(), ex, testCode)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != TestFailure {
		t.Fatalf("Status = %s, want %s\n%s", result.Status, TestFailure, result.Output)
	}
}

func signExercise() exercise.Exercise {
	return exercise.Exercise{
		ID:       "go-007",
		Language: "go",
		Kind:     exercise.KindTestWriting,
		SubjectCode: `package exercise

func Sign(n int) string {
	if n > 0 {
		return "positive"
	}
	if n < 0 {
		return "negative"
	}
	return "zero"
}
`,
		Mutants: []string{
			`package exercise

func Sign(n int) string {
	if n >= 0 {
		return "positive"
	}
	return "negative"
}
`,
			`package exercise

func Sign(n int) string {
	if n > 0 {
		return "positive"
	}
	return "negative"
}
`,
		},
		MutantHints: []string{
			"misreports zero as positive instead of zero",
			"misreports zero as negative instead of zero",
		},
	}
}

func addExercise() exercise.Exercise {
	return exercise.Exercise{
		ID:       "go-add",
		Language: "go",
		Tests: `package exercise

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Fatalf("got %d", got)
	}
}
`,
	}
}
