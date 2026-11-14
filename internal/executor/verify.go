package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code-muscle-memory/internal/exercise"
)

// VerifyAuthoring runs the checks on an exercise that schema validation
// cannot: it compiles and executes the exercise's own code. It returns the
// list of problems found, empty when the exercise is sound; the error is
// reserved for infrastructure failures.
//
// Implementation exercises must carry a solution that passes the hidden
// tests, and their starter code must not pass them (tests a blank stub
// satisfies verify nothing). Test-writing exercises must have a compiling
// subject and compiling, non-identical mutants — a mutant that does not
// compile is never exercised by grading.
func (g GoExecutor) VerifyAuthoring(ctx context.Context, ex exercise.Exercise) ([]string, error) {
	if ex.EffectiveKind() == exercise.KindTestWriting {
		return g.verifyTestWritingAuthoring(ctx, ex)
	}
	return g.verifyImplementationAuthoring(ctx, ex)
}

func (g GoExecutor) verifyImplementationAuthoring(ctx context.Context, ex exercise.Exercise) ([]string, error) {
	var problems []string
	if strings.TrimSpace(ex.Solution) == "" {
		problems = append(problems, "has no solution, so the hidden tests cannot be verified; add one")
	} else {
		result, err := g.Evaluate(ctx, ex, ex.Solution)
		if err != nil {
			return nil, err
		}
		if result.Status != Success {
			problems = append(problems, "solution does not pass the hidden tests:\n"+result.Output)
		}
	}

	if strings.TrimSpace(ex.StarterCode) != "" {
		result, err := g.Evaluate(ctx, ex, ex.StarterCode)
		if err != nil {
			return nil, err
		}
		if result.Status == Success {
			problems = append(problems, "starter code already passes the hidden tests, so they verify nothing")
		}
	}
	return problems, nil
}

func (g GoExecutor) verifyTestWritingAuthoring(ctx context.Context, ex exercise.Exercise) ([]string, error) {
	var problems []string
	compiles, output, err := compilesAlone(ctx, ex.SubjectCode)
	if err != nil {
		return nil, err
	}
	if !compiles {
		problems = append(problems, "subject code does not compile:\n"+output)
	}

	for i, mutant := range ex.Mutants {
		if mutant == ex.SubjectCode {
			problems = append(problems, fmt.Sprintf("mutant %d is identical to the subject code", i))
			continue
		}
		compiles, output, err := compilesAlone(ctx, mutant)
		if err != nil {
			return nil, err
		}
		if !compiles {
			problems = append(problems, fmt.Sprintf("mutant %d does not compile:\n%s", i, output))
		}
	}
	return problems, nil
}

// compilesAlone builds source as the only file of a scratch module and
// reports whether it compiled, with the compiler output when it did not.
func compilesAlone(ctx context.Context, source string) (bool, string, error) {
	dir, err := os.MkdirTemp("", "cmm-go-*")
	if err != nil {
		return false, "", err
	}
	defer os.RemoveAll(dir)

	files := map[string]string{
		"go.mod":     "module exercise\n\ngo 1.22\n",
		"subject.go": source,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return false, "", err
		}
	}

	output, err := runGo(ctx, dir, "build", "./...")
	if err != nil {
		return false, output, nil
	}
	return true, "", nil
}
