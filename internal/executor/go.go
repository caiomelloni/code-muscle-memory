package executor

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"code-muscle-memory/internal/exercise"
)

type Status string

const (
	CompileError Status = "compile_error"
	TestFailure  Status = "test_failure"
	Success      Status = "success"
)

type Result struct {
	Status Status
	Output string
}

type GoExecutor struct{}

func NewGoExecutor() GoExecutor {
	return GoExecutor{}
}

func (GoExecutor) Evaluate(ctx context.Context, ex exercise.Exercise, solution string) (Result, error) {
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

func runGo(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}
