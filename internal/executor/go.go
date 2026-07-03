package executor

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code-muscle-memory/internal/exercise"
)

type Status string

const (
	CompileError   Status = "compile_error"
	TestFailure    Status = "test_failure"
	MutantEscaped  Status = "mutant_escaped"
	MissingFeature Status = "missing_feature"
	Success        Status = "success"
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

	if feature, ok := missingTestFeature(ex.RequiredTestFeatures, testCode); ok {
		return Result{Status: MissingFeature, Output: featureMessage(feature)}, nil
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

// missingTestFeature reports the first required structural feature the user's
// test does not exhibit. Detection is heuristic: it looks at the shape of the
// test functions, not at what they execute.
func missingTestFeature(required []string, testCode string) (string, bool) {
	if len(required) == 0 {
		return "", false
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "solution_test.go", testCode, 0)
	if err != nil {
		// The code already compiled; an unparsable file should not happen.
		return "", false
	}

	tests := testFuncs(file)
	for _, feature := range required {
		found := false
		switch feature {
		case exercise.FeatureSubtests:
			found = anyTestFunc(tests, hasSubtestCall)
		case exercise.FeatureTable:
			found = anyTestFunc(tests, hasRangeLoop)
		}
		if !found {
			return feature, true
		}
	}
	return "", false
}

type testFunc struct {
	decl      *ast.FuncDecl
	paramName string
}

// testFuncs returns the file's test functions: no receiver, name starting
// with "Test", and a single pointer parameter (the *testing.T).
func testFuncs(file *ast.File) []testFunc {
	var tests []testFunc
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		params := fn.Type.Params
		if params == nil || len(params.List) != 1 || len(params.List[0].Names) != 1 {
			continue
		}
		if _, ok := params.List[0].Type.(*ast.StarExpr); !ok {
			continue
		}
		tests = append(tests, testFunc{decl: fn, paramName: params.List[0].Names[0].Name})
	}
	return tests
}

func anyTestFunc(tests []testFunc, check func(testFunc) bool) bool {
	for _, test := range tests {
		if check(test) {
			return true
		}
	}
	return false
}

func hasSubtestCall(test testFunc) bool {
	found := false
	ast.Inspect(test.decl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Run" {
			return true
		}
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == test.paramName {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasRangeLoop(test testFunc) bool {
	found := false
	ast.Inspect(test.decl.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.RangeStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func featureMessage(feature string) string {
	switch feature {
	case exercise.FeatureSubtests:
		return "Your test passes, but it doesn't run each case as a named subtest. Use the testing.T method called \"Run\" to give each case its own name."
	case exercise.FeatureTable:
		return "Your test passes, but it doesn't drive its cases from a table. Collect the cases in a data structure and loop over them."
	default:
		return "Your test passes, but it doesn't have the structure this exercise asks for. Re-read the instructions and reshape it."
	}
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
