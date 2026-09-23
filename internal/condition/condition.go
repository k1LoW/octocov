// Package condition evaluates the conditions written in the config (`if:`, `acceptable:` and the
// like). Conditions are CEL expressions. Conditions written for expr-lang/expr, which octocov
// used to evaluate them with, are still evaluated by it with a deprecation warning.
package condition

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sync"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"
	"cel.dev/cel-go/common/types"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var (
	warnOut io.Writer = os.Stderr
	warned  sync.Map
)

// Program is a compiled condition.
type Program struct {
	src  string
	cel  cel.Program
	expr *vm.Program
	// celErr is why the condition could not be compiled as CEL, if it could not.
	celErr error
}

// Compile compiles a condition. vars holds the variables the condition refers to, and only the
// types of their values are read, so zero values do.
func Compile(src string, vars map[string]any) (*Program, error) {
	p := &Program{src: src}
	prg, err := compileCEL(src, vars)
	if err == nil {
		p.cel = prg
		return p, nil
	}
	p.celErr = err
	if err := p.compileExpr(); err != nil {
		// CEL is the language conditions are written in, so its error is the one to fix.
		return nil, p.celErr
	}
	return p, nil
}

// Eval compiles and evaluates a condition once.
func Eval(src string, vars map[string]any) (bool, error) {
	p, err := Compile(src, vars)
	if err != nil {
		return false, err
	}
	return p.Eval(vars)
}

// Eval evaluates the condition with vars.
func (p *Program) Eval(vars map[string]any) (bool, error) {
	celErr := p.celErr
	if p.cel != nil {
		out, _, err := p.cel.Eval(celVars(vars))
		if err == nil {
			if b, ok := out.Value().(bool); ok {
				return b, nil
			}
			err = fmt.Errorf("invalid condition `%s`: it is evaluated to %v, not a boolean", p.src, out.Value())
		}
		celErr = err
	}
	if p.expr == nil {
		if err := p.compileExpr(); err != nil {
			return false, celErr
		}
	}
	v, err := expr.Run(p.expr, vars)
	if err != nil {
		return false, celErr
	}
	b, ok := v.(bool)
	if !ok {
		return false, celErr
	}
	warn(p.src, celErr)
	return b, nil
}

func (p *Program) compileExpr() error {
	prg, err := expr.Compile(fmt.Sprintf("(%s) == true", p.src))
	if err != nil {
		return err
	}
	p.expr = prg
	return nil
}

func compileCEL(src string, vars map[string]any) (cel.Program, error) {
	opts := []cel.EnvOption{
		cel.CrossTypeNumericComparisons(true),
		cel.OptionalTypes(),
	}
	for k, v := range vars {
		t := cel.DynType
		if isInteger(v) {
			t = cel.IntType
		}
		opts = append(opts, cel.Variable(k, t))
	}
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, err
	}
	typed, err := probeTypes(env, src)
	if err != nil {
		return nil, err
	}
	a, iss := env.Parse(src)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	rewrite := map[int64]struct{}{}
	classify(ast.NavigateAST(a.NativeRep()), typed, rewrite)
	a, iss = parseRewritten(env, src, rewrite)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	checked, iss := env.Check(a)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	return env.Program(checked)
}

// probeTypes checks src with every int literal wrapped in dyn(), for the types of everything
// else in it. The literals are what is being decided, and checked as they are written they can
// fail the check on their own, as `7 / 2 == 3.5` does.
func probeTypes(env *cel.Env, src string) (*ast.AST, error) {
	a, iss := env.Parse(src)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	root := ast.NavigateAST(a.NativeRep())
	var maxID int64
	for _, e := range ast.MatchDescendants(root, func(ast.NavigableExpr) bool { return true }) {
		maxID = max(maxID, e.ID())
	}
	fac := ast.NewExprFactory()
	for _, e := range ast.MatchDescendants(root, ast.ConstantValueMatcher()) {
		v, ok := e.AsLiteral().(types.Int)
		if !ok {
			continue
		}
		maxID++
		e.SetKindCase(fac.NewCall(e.ID(), overloads.TypeConvertDyn, fac.NewLiteral(maxID, v)))
	}
	checked, iss := env.Check(a)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	return checked.NativeRep(), nil
}

// parseRewritten parses src with the int literals of rewrite turned into doubles. Every rewrite
// starts from a fresh parse rather than from an AST that has been checked, which the checker
// may share with its result. A parse of the same source numbers the nodes the same way, so the
// ids found on one parse apply to the next.
func parseRewritten(env *cel.Env, src string, rewrite map[int64]struct{}) (*cel.Ast, *cel.Issues) {
	a, iss := env.Parse(src)
	if iss.Err() != nil {
		return nil, iss
	}
	fac := ast.NewExprFactory()
	for _, e := range ast.MatchDescendants(ast.NavigateAST(a.NativeRep()), ast.ConstantValueMatcher()) {
		if _, ok := rewrite[e.ID()]; !ok {
			continue
		}
		if v, ok := e.AsLiteral().(types.Int); ok {
			e.SetKindCase(fac.NewLiteral(e.ID(), types.Double(v)))
		}
	}
	return a, iss
}

// numClass is what an expression is as a number, for deciding its int literals by.
type numClass int

const (
	notNumeric numClass = iota
	// intLiteral is made of int literals alone, which can be rewritten into doubles.
	intLiteral
	// intTyped is an int that cannot be rewritten, such as size() or an integer variable.
	intTyped
	// doubleTyped is a double, or a dyn, which the measured values are declared as.
	doubleTyped
)

var (
	arithmeticOperators = map[string]struct{}{
		operators.Add:      {},
		operators.Subtract: {},
		operators.Multiply: {},
		operators.Divide:   {},
	}
	comparisonOperators = map[string]struct{}{
		operators.Less:          {},
		operators.LessEquals:    {},
		operators.Greater:       {},
		operators.GreaterEquals: {},
		operators.Equals:        {},
		operators.NotEquals:     {},
	}
)

// classify adds to rewrite the int literals of e to be rewritten into doubles, and returns what
// e is as a number. Measured values are float64 while thresholds are mostly written as integers,
// and CEL has no arithmetic between int and double, so `current > prev + 1` would not evaluate
// otherwise. Overloading the arithmetic operators for mixed operands is not an option since the
// standard ones are singleton functions. The literals are rewritten a subexpression at a time,
// where one made of int literals alone meets a double under an arithmetic or comparison
// operator, so that `current + (1 + 1)` is rewritten as a whole while `hour + 1`, an index or
// an operand of `%` keeps its ints.
func classify(e ast.NavigableExpr, typed *ast.AST, rewrite map[int64]struct{}) numClass {
	children := e.Children()
	switch e.Kind() {
	case ast.LiteralKind:
		switch e.AsLiteral().(type) {
		case types.Int:
			return intLiteral
		case types.Double:
			return doubleTyped
		default:
			return notNumeric
		}
	case ast.CallKind:
		fn := e.AsCall().FunctionName()
		if _, ok := arithmeticOperators[fn]; ok && len(children) == 2 {
			return classifyPair(children[0], children[1], typed, rewrite)
		}
		if _, ok := comparisonOperators[fn]; ok && len(children) == 2 {
			classifyPair(children[0], children[1], typed, rewrite)
			return notNumeric
		}
		if fn == operators.Negate && len(children) == 1 {
			return classify(children[0], typed, rewrite)
		}
		if fn == operators.Modulo {
			for _, c := range children {
				classify(c, typed, rewrite)
			}
			return intTyped
		}
	}
	for _, c := range children {
		classify(c, typed, rewrite)
	}
	switch typed.GetType(e.ID()).Kind() {
	case types.IntKind, types.UintKind:
		return intTyped
	case types.DoubleKind, types.DynKind:
		return doubleTyped
	default:
		return notNumeric
	}
}

func classifyPair(l, r ast.NavigableExpr, typed *ast.AST, rewrite map[int64]struct{}) numClass {
	lc := classify(l, typed, rewrite)
	rc := classify(r, typed, rewrite)
	switch {
	case lc == doubleTyped && rc == intLiteral:
		toDouble(r, rewrite)
		return doubleTyped
	case lc == intLiteral && rc == doubleTyped:
		toDouble(l, rewrite)
		return doubleTyped
	case lc == doubleTyped || rc == doubleTyped:
		return doubleTyped
	case lc == intTyped || rc == intTyped:
		return intTyped
	case lc == intLiteral && rc == intLiteral:
		return intLiteral
	default:
		return notNumeric
	}
}

func toDouble(e ast.NavigableExpr, rewrite map[int64]struct{}) {
	for _, l := range ast.MatchDescendants(e, ast.ConstantValueMatcher()) {
		if _, ok := l.AsLiteral().(types.Int); ok {
			rewrite[l.ID()] = struct{}{}
		}
	}
}

func isInteger(v any) bool {
	switch reflect.ValueOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// celVars converts the integer variables into int64, the one integer type CEL takes, so that a
// named one such as time.Month is read as the int it has been declared as.
func celVars(vars map[string]any) map[string]any {
	converted := make(map[string]any, len(vars))
	for k, v := range vars {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			converted[k] = rv.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			converted[k] = int64(rv.Uint()) // #nosec G115
		default:
			converted[k] = v
		}
	}
	return converted
}

// warn tells once per condition that it is evaluated by expr-lang/expr. It is written to stderr
// rather than to the log, which is discarded unless DEBUG is set.
func warn(src string, celErr error) {
	if _, loaded := warned.LoadOrStore(src, struct{}{}); loaded {
		return
	}
	if _, err := fmt.Fprintf(warnOut, "Deprecated: the condition `%s` is evaluated as an expr-lang/expr expression, which will not be supported in a future release. Rewrite it as a CEL expression: %v\n", src, celErr); err != nil {
		// The condition has been evaluated all the same, and failing a coverage gate because
		// stderr could not be written to would be worse than losing the warning.
		log.Printf("failed to write the deprecation warning: %v", err)
	}
}
