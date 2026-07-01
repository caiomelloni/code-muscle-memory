package executor

import (
	"context"
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
