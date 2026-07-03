package app

import (
	"bufio"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"code-muscle-memory/internal/executor"
	"code-muscle-memory/internal/exercise"
	"code-muscle-memory/internal/scheduler"
	"code-muscle-memory/internal/storage"
)

type Config struct {
	ExerciseDir  string
	ProgressPath string
	Stdout       io.Writer
	Stderr       io.Writer
	Stdin        io.Reader
	Now          func() time.Time
}

type App struct {
	cfg       Config
	store     storage.Store
	scheduler scheduler.Scheduler
	executor  executor.GoExecutor
}

func New(cfg Config) App {
	if cfg.Stdout == nil {
		cfg.Stdout = io.Discard
	}
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return App{
		cfg:       cfg,
		store:     storage.NewJSONStore(cfg.ProgressPath),
		scheduler: scheduler.NewSM2(),
		executor:  executor.NewGoExecutor(),
	}
}

func (a App) Next(ctx context.Context) error {
	exercises, progress, err := a.load()
	if err != nil {
		return err
	}

	now := a.cfg.Now()
	ex, ok := chooseNext(exercises, progress, now)
	if !ok {
		fmt.Fprintln(a.cfg.Stdout, "No exercises available.")
		return nil
	}
	item := progress.Items[ex.ID]
	item.ExerciseID = ex.ID
	progress.CurrentExerciseID = ex.ID
	if err := a.store.Save(progress); err != nil {
		return err
	}

	if !item.DueAt.IsZero() && item.DueAt.After(now) {
		fmt.Fprintf(a.cfg.Stdout, "Next review is not due yet. Earliest: %s\n", item.DueAt.Format(time.RFC1123))
		return nil
	}

	solutionPath, cleanup, err := writeStarter(ex)
	if err != nil {
		return err
	}
	defer cleanup()

	a.printExercise(ex, solutionPath)
	if err := openEditor(ctx, solutionPath); err != nil {
		return err
	}

	solution, err := os.ReadFile(solutionPath)
	if err != nil {
		return err
	}
	result, err := a.executor.Evaluate(ctx, ex, string(solution))
	if err != nil {
		return err
	}

	rating := scheduler.Good
	switch result.Status {
	case executor.Success:
		fmt.Fprintln(a.cfg.Stdout, "\nPASS")
		if strings.TrimSpace(result.Output) != "" {
			fmt.Fprintln(a.cfg.Stdout, result.Output)
		}
		rating = a.promptRating()
	case executor.CompileError:
		fmt.Fprintln(a.cfg.Stdout, "\nCOMPILE ERROR")
		fmt.Fprintln(a.cfg.Stdout, result.Output)
		return nil
	case executor.TestFailure:
		fmt.Fprintln(a.cfg.Stdout, "\nTEST FAILURE")
		fmt.Fprintln(a.cfg.Stdout, result.Output)
		return nil
	}

	progress.CurrentExerciseID = ""
	progress.Items[ex.ID] = a.scheduler.Review(now, item, rating)
	return a.store.Save(progress)
}

func (a App) List(ctx context.Context) error {
	_ = ctx
	exercises, progress, err := a.load()
	if err != nil {
		return err
	}
	now := a.cfg.Now()
	for _, ex := range exercises {
		item, ok := progress.Items[ex.ID]
		status := "new"
		if ok {
			if item.DueAt.After(now) {
				status = "due " + item.DueAt.Format("2006-01-02")
			} else {
				status = "due now"
			}
		}
		fmt.Fprintf(a.cfg.Stdout, "%-22s %-12s difficulty=%d topic=%s\n", ex.ID, status, ex.Difficulty, ex.Topic)
	}
	return nil
}

func (a App) Stats(ctx context.Context) error {
	_ = ctx
	exercises, progress, err := a.load()
	if err != nil {
		return err
	}
	now := a.cfg.Now()
	reviewed := 0
	due := 0
	for _, ex := range exercises {
		item, ok := progress.Items[ex.ID]
		if !ok {
			due++
			continue
		}
		if item.ReviewCount > 0 {
			reviewed++
		}
		if !item.DueAt.After(now) {
			due++
		}
	}
	fmt.Fprintf(a.cfg.Stdout, "Exercises: %d\nReviewed: %d\nDue now: %d\n", len(exercises), reviewed, due)
	return nil
}

func (a App) load() ([]exercise.Exercise, storage.ProgressFile, error) {
	exercises, err := exercise.LoadDir(a.cfg.ExerciseDir)
	if err != nil {
		return nil, storage.ProgressFile{}, err
	}
	progress, err := a.store.Load()
	if err != nil {
		return nil, storage.ProgressFile{}, err
	}
	return exercises, progress, nil
}

func (a App) printExercise(ex exercise.Exercise, solutionPath string) {
	fmt.Fprintf(a.cfg.Stdout, "\n# %s\n\n%s\n\nObjective: %s\nTopic: %s | Difficulty: %d\n\nEditing: %s\n", ex.Title, ex.Description, ex.Objective, ex.Topic, ex.Difficulty, solutionPath)
}

func (a App) promptRating() scheduler.Rating {
	fmt.Fprint(a.cfg.Stdout, "Rate this review [good/easy/hard] (default good): ")
	reader := bufio.NewReader(a.cfg.Stdin)
	text, _ := reader.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "easy", "e":
		return scheduler.Easy
	case "hard", "h":
		return scheduler.Hard
	default:
		return scheduler.Good
	}
}

func chooseNext(exercises []exercise.Exercise, progress storage.ProgressFile, now time.Time) (exercise.Exercise, bool) {
	if len(exercises) == 0 {
		return exercise.Exercise{}, false
	}
	if progress.CurrentExerciseID != "" {
		for _, ex := range exercises {
			if ex.ID == progress.CurrentExerciseID {
				return ex, true
			}
		}
	}

	items := progress.Items
	ordered := append([]exercise.Exercise(nil), exercises...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, leftOK := items[ordered[i].ID]
		right, rightOK := items[ordered[j].ID]
		if !leftOK && rightOK {
			return true
		}
		if leftOK && !rightOK {
			return false
		}
		if leftOK && rightOK && !left.DueAt.Equal(right.DueAt) {
			return left.DueAt.Before(right.DueAt)
		}
		if ordered[i].Difficulty != ordered[j].Difficulty {
			return ordered[i].Difficulty < ordered[j].Difficulty
		}
		return ordered[i].ID < ordered[j].ID
	})
	for _, ex := range ordered {
		item, ok := items[ex.ID]
		if !ok || !item.DueAt.After(now) {
			return ex, true
		}
	}
	return ordered[0], true
}

func writeStarter(ex exercise.Exercise) (string, func(), error) {
	dir, err := os.MkdirTemp("", "cmm-edit-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "solution.go")
	code := starterWithInstructions(ex)
	if strings.TrimSpace(code) == "" {
		code = "package exercise\n"
	}
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

func starterWithInstructions(ex exercise.Exercise) string {
	lines := []string{
		"// " + ex.Title,
		"//",
		"// What to do: " + ex.Description,
	}
	if strings.TrimSpace(ex.Objective) != "" {
		lines = append(lines, "// Objective: "+ex.Objective)
	}
	for _, target := range implementationTargets(ex.StarterCode) {
		lines = append(lines, "//")
		lines = appendCommentBlock(lines, "Implement: ", target)
	}
	lines = append(lines, "", "package exercise", "")
	return strings.Join(lines, "\n")
}

func implementationTargets(starter string) []string {
	if strings.TrimSpace(starter) == "" {
		return nil
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "starter.go", starter, parser.ParseComments)
	if err != nil {
		return nil
	}

	var targets []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				targets = append(targets, typeDescriptions(d)...)
			}
		case *ast.FuncDecl:
			targets = append(targets, funcDescription(d))
		}
	}
	return targets
}

func typeDescriptions(decl *ast.GenDecl) []string {
	var descriptions []string
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			descriptions = append(descriptions, "a type called "+quoted(typeSpec.Name.Name))
			continue
		}

		fields := fieldDescriptions(structType.Fields, "field")
		if len(fields) == 0 {
			descriptions = append(descriptions, "a struct type called "+quoted(typeSpec.Name.Name))
			continue
		}
		descriptions = append(descriptions, "a struct type called "+quoted(typeSpec.Name.Name)+" with "+joinEnglish(fields))
	}
	return descriptions
}

func funcDescription(fn *ast.FuncDecl) string {
	subject := "a function called " + quoted(fn.Name.Name)
	if fn.Recv == nil {
		return subject + ", " + paramsDescription(fn.Type.Params) + ", and " + resultsDescription(fn.Type.Results)
	}
	return "a method called " + quoted(fn.Name.Name) + " on " + receiverDescription(fn.Recv) + ", " + paramsDescription(fn.Type.Params) + ", and " + resultsDescription(fn.Type.Results)
}

func receiverDescription(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return "the receiver type"
	}
	return typePhrase(recv.List[0].Type)
}

func paramsDescription(params *ast.FieldList) string {
	fields := fieldDescriptions(params, "parameter")
	if len(fields) == 0 {
		return "receives no parameters"
	}
	return "receives " + joinEnglish(fields)
}

func resultsDescription(results *ast.FieldList) string {
	fields := fieldDescriptions(results, "value")
	if len(fields) == 0 {
		return "returns nothing"
	}
	return "returns " + joinEnglish(fields)
}

func fieldDescriptions(fields *ast.FieldList, namedKind string) []string {
	if fields == nil {
		return nil
	}

	var descriptions []string
	for _, field := range fields.List {
		fieldType := typePhrase(field.Type)
		if len(field.Names) == 0 {
			descriptions = append(descriptions, article(fieldType)+" "+fieldType)
			continue
		}
		for _, name := range field.Names {
			descriptions = append(descriptions, "a "+namedKind+" called "+quoted(name.Name)+" of type "+fieldType)
		}
	}
	return descriptions
}

func quoted(name string) string {
	return "\"" + name + "\""
}

func typePhrase(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "slice of " + typePhrase(t.Elt)
		}
		return "array of " + typePhrase(t.Elt)
	case *ast.StarExpr:
		return "pointer to " + typePhrase(t.X)
	case *ast.SelectorExpr:
		return typePhrase(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map from " + typePhrase(t.Key) + " to " + typePhrase(t.Value)
	case *ast.InterfaceType:
		return "interface"
	case *ast.StructType:
		return "struct"
	default:
		return nodeString(token.NewFileSet(), expr)
	}
}

func joinEnglish(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

func article(phrase string) string {
	lower := strings.ToLower(phrase)
	for _, prefix := range []string{"a", "e", "i", "o", "u"} {
		if strings.HasPrefix(lower, prefix) {
			return "an"
		}
	}
	return "a"
}

func nodeString(fileSet *token.FileSet, node any) string {
	var builder strings.Builder
	_ = printer.Fprint(&builder, fileSet, node)
	return builder.String()
}

func appendCommentBlock(lines []string, prefix, text string) []string {
	for i, line := range strings.Split(text, "\n") {
		if i == 0 {
			lines = append(lines, "// "+prefix+line)
			continue
		}
		lines = append(lines, "// "+line)
	}
	return lines
}

func openEditor(ctx context.Context, path string) error {
	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", editor+" "+shellQuote(path))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
