package executor

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code-muscle-memory/internal/exercise"
)

type Status string

const (
	CompileError  Status = "compile_error"
	TestFailure   Status = "test_failure"
	MutantEscaped Status = "mutant_escaped"
	Success       Status = "success"
)

type Result struct {
	Status Status
	Output string
}

type GoExecutor struct{}

func NewGoExecutor() GoExecutor {
	return GoExecutor{}
}

func (g GoExecutor) Evaluate(ctx context.Context, ex exercise.Exercise, solution string) (Result, error) {
	if ex.EffectiveKind() == exercise.KindTestWriting {
		return g.evaluateTestWriting(ctx, ex, solution)
	}
	return g.evaluateImplementation(ctx, ex, solution)
}

func (GoExecutor) evaluateImplementation(ctx context.Context, ex exercise.Exercise, solution string) (Result, error) {
	dir, err := os.MkdirTemp("", "cmm-go-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)

	files := map[string]string{
		"go.mod":           "module exercise\n\ngo 1.22\n",
		"solution.go":      solution,
		"solution_test.go": ex.Tests,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return Result{}, err
		}
	}

	compileOutput, err := runGo(ctx, dir, "test", "-run", "^$")
	if err != nil {
		return Result{Status: CompileError, Output: compileOutput}, nil
	}

	testOutput, err := runGo(ctx, dir, "test", "./...")
	if err != nil {
		return Result{Status: TestFailure, Output: testOutput}, nil
	}
	return Result{Status: Success, Output: testOutput}, nil
}

// evaluateTestWriting grades an exercise where the user writes the test and
// the exercise ships the correct implementation. The user's test must pass
// against ex.SubjectCode and fail against every entry in ex.Mutants, proving
// it actually verifies the required behavior rather than asserting nothing.
func (GoExecutor) evaluateTestWriting(ctx context.Context, ex exercise.Exercise, testCode string) (Result, error) {
	dir, err := os.MkdirTemp("", "cmm-go-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module exercise\n\ngo 1.22\n"), 0o644); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "solution_test.go"), []byte(testCode), 0o644); err != nil {
		return Result{}, err
	}
	writeSubject := func(subject string) error {
		return os.WriteFile(filepath.Join(dir, "subject.go"), []byte(subject), 0o644)
	}

	if err := writeSubject(ex.SubjectCode); err != nil {
		return Result{}, err
	}
	compileOutput, err := runGo(ctx, dir, "test", "-run", "^$")
	if err != nil {
		return Result{Status: CompileError, Output: compileOutput}, nil
	}
	testOutput, err := runGo(ctx, dir, "test", "./...")
	if err != nil {
		return Result{Status: TestFailure, Output: testOutput}, nil
	}

	for i, mutant := range ex.Mutants {
		if err := writeSubject(mutant); err != nil {
			return Result{}, err
		}
		if _, err := runGo(ctx, dir, "test", "./..."); err == nil {
			return Result{Status: MutantEscaped, Output: mutantEscapeMessage(ex, i)}, nil
		}
	}

	return Result{Status: Success, Output: testOutput}, nil
}

func mutantEscapeMessage(ex exercise.Exercise, index int) string {
	if index < len(ex.MutantHints) && strings.TrimSpace(ex.MutantHints[index]) != "" {
		return "Your test passed even though the implementation " + ex.MutantHints[index] + ". Add a case that would catch this."
	}
	return "Your test passed against a buggy implementation, meaning it doesn't fully verify the required behavior. Try adding another case."
}

func runGo(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}
