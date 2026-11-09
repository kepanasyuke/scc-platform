package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxStatements = 2
	wideDirective = "style:wide"
)

var skippedDirs = map[string]bool{
	"vendor":       true,
	"node_modules": true,
	".next":        true,
	"dist":         true,
	"cmd":          true,
	"migrations":   true,
	"deploy":       true,
	"tools":        true,
	"testdata":     true,
}

type finding struct {
	path    string
	line    int
	message string
}

type scanner struct {
	fset   *token.FileSet
	path   string
	source []string
	found  []finding
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	found, err := scanTree(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "stylecheck:", err)
		os.Exit(2)
	}
	printFindings(found)
	fmt.Printf("stylecheck: %d violation(s) in %s\n", len(found), root)
	if len(found) > 0 {
		os.Exit(1)
	}
}

func scanTree(root string) ([]finding, error) {
	collected := make([]finding, 0, 64)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		return visit(&collected, path, entry, walkErr)
	})
	return collected, err
}

func visit(collected *[]finding, path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() && skippedDirs[entry.Name()] {
		return filepath.SkipDir
	}
	if entry.IsDir() || !strings.HasSuffix(path, ".go") {
		return nil
	}
	if strings.HasSuffix(path, "_test.go") {
		return nil
	}
	found, err := inspect(path)
	*collected = append(*collected, found...)
	return err
}

func inspect(path string) ([]finding, error) {
	raw, err := os.ReadFile(path)
	fset := token.NewFileSet()
	file, parseErr := parser.ParseFile(fset, path, raw, parser.ParseComments)
	scanner := &scanner{fset: fset, path: path, source: strings.Split(string(raw), "\n"), found: make([]finding, 0, 4)}
	ast.Inspect(file, scanner.visitNode)
	return scanner.found, firstErr(err, parseErr)
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *scanner) visitNode(node ast.Node) bool {
	decl, ok := node.(*ast.FuncDecl)
	if ok && decl.Body != nil && !s.wide(decl) {
		s.checkFunction(decl)
	}
	return true
}

func (s *scanner) wide(decl *ast.FuncDecl) bool {
	start := max(0, s.fset.Position(decl.Pos()).Line-2)
	end := s.fset.Position(decl.End()).Line
	return strings.Contains(strings.Join(s.source[start:end], "\n"), wideDirective)
}

func (s *scanner) checkFunction(decl *ast.FuncDecl) {
	statements := logic(decl.Body.List)
	s.add(len(statements) > maxStatements, decl.Pos(), fmt.Sprintf(
		"%s: %d statements, максимум %d (объявите декомпозицию или %s)", decl.Name.Name, len(statements), maxStatements, wideDirective))
	s.checkReturns(decl, statements)
	ast.Inspect(decl.Body, s.visitStatement)
}

func logic(statements []ast.Stmt) []ast.Stmt {
	guarded := len(statements) >= 2 && isLock(statements[0]) && isUnlock(statements[1])
	if guarded {
		return statements[2:]
	}
	return statements
}

func isLock(stmt ast.Stmt) bool {
	call := calledName(stmt)
	return call == "Lock" || call == "RLock"
}

func isUnlock(stmt ast.Stmt) bool {
	deferStmt, isDefer := stmt.(*ast.DeferStmt)
	name := ""
	if isDefer {
		name = callName(deferStmt.Call)
	}
	return name == "Unlock" || name == "RUnlock"
}

func calledName(stmt ast.Stmt) string {
	expression, isExpression := stmt.(*ast.ExprStmt)
	name := ""
	if isExpression {
		name = callName(expression.X)
	}
	return name
}

func callName(expression ast.Expr) string {
	call, isCall := expression.(*ast.CallExpr)
	name := ""
	if isCall {
		selector, isSelector := call.Fun.(*ast.SelectorExpr)
		if isSelector {
			name = selector.Sel.Name
		}
	}
	return name
}

func (s *scanner) checkReturns(decl *ast.FuncDecl, statements []ast.Stmt) {
	returns := collectReturns(decl.Body)
	position := decl.Pos()
	if len(returns) > 0 {
		position = returns[0].Pos()
	}
	s.add(len(returns) > 1, position, fmt.Sprintf("%s: %d return, СЂР°Р·СЂРµС€С‘РЅ СЂРѕРІРЅРѕ РѕРґРёРЅ РІС‹С…РѕРґ", decl.Name.Name, len(returns)))
	s.add(len(returns) == 1 && !isLast(statements, returns[0]), position, "return РґРѕР»Р¶РµРЅ Р±С‹С‚СЊ РїРѕСЃР»РµРґРЅРёРј РѕРїРµСЂР°С‚РѕСЂРѕРј С„СѓРЅРєС†РёРё")
}

func collectReturns(body *ast.BlockStmt) []*ast.ReturnStmt {
	found := make([]*ast.ReturnStmt, 0, 2)
	ast.Inspect(body, func(node ast.Node) bool {
		_, isLiteral := node.(*ast.FuncLit)
		stmt, isReturn := node.(*ast.ReturnStmt)
		found = appendReturn(found, stmt, isReturn && !isLiteral)
		return !isLiteral
	})
	return found
}

func appendReturn(found []*ast.ReturnStmt, stmt *ast.ReturnStmt, ok bool) []*ast.ReturnStmt {
	return pick(ok, append(found, stmt), found)
}

func pick[T any](ok bool, yes, no T) T {
	result := no
	if ok {
		result = yes
	}
	return result
}

func isLast(statements []ast.Stmt, target ast.Node) bool {
	last := ast.Node(nil)
	if len(statements) > 0 {
		last = statements[len(statements)-1]
	}
	return last != nil && last == target
}

func (s *scanner) visitStatement(node ast.Node) bool {
	branch, isBranch := node.(*ast.BranchStmt)
	loop, isLoop := node.(*ast.ForStmt)
	s.reportBranch(branch, isBranch)
	s.reportWhile(loop, isLoop)
	return true
}

func (s *scanner) reportBranch(branch *ast.BranchStmt, isBranch bool) {
	forbidden := isBranch && branch.Tok != token.RANGE
	if forbidden {
		s.add(true, branch.Pos(), fmt.Sprintf("РѕРїРµСЂР°С‚РѕСЂ %s Р·Р°РїСЂРµС‰С‘РЅ СЃС‚РёР»РµРј РїСЂРѕРµРєС‚Р°", branch.Tok))
	}
}

func (s *scanner) reportWhile(loop *ast.ForStmt, isLoop bool) {
	condition := isLoop && loop.Init == nil && loop.Post == nil && loop.Cond != nil
	if condition {
		s.add(true, loop.Pos(), "С†РёРєР» РІ СЃС‚РёР»Рµ while Р·Р°РїСЂРµС‰С‘РЅ: РёСЃРїРѕР»СЊР·СѓР№С‚Рµ range РїРѕ СЃСЂРµР·Сѓ РёР»Рё РєР°РЅР°Р»Сѓ")
	}
}

func (s *scanner) add(condition bool, pos token.Pos, message string) {
	if condition {
		s.found = append(s.found, finding{path: s.path, line: s.fset.Position(pos).Line, message: message})
	}
}

func printFindings(found []finding) {
	sort.SliceStable(found, func(i, j int) bool {
		return lessFinding(found[i], found[j])
	})
	for _, item := range found {
		fmt.Printf("%s:%d: %s\n", item.path, item.line, item.message)
	}
}

func lessFinding(left, right finding) bool {
	ordered := left.path < right.path || (left.path == right.path && left.line < right.line)
	return ordered
}
