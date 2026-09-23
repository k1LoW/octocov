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
	rewrite := map[int64]struct{}{}
	a, iss := parseRewritten(env, src, rewrite)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	typed, iss := env.Check(a)
	if iss.Err() != nil {
		// Literals alone can fail the check, as `7 / 2 == 3.5` does. Only the subexpressions
		// made of literals are rewritten for it, as rewriting the rest here, with no types to go
		// by, would turn the literal of `hour + 1` into a double beside an int.
		literalOnlyIntLiterals(a.NativeRep(), rewrite)
		a, iss = parseRewritten(env, src, rewrite)
		if iss.Err() != nil {
			return nil, iss.Err()
		}
		typed, iss = env.Check(a)
		if iss.Err() != nil {
			return nil, iss.Err()
		}
	}
	intLiterals(a.NativeRep(), typed.NativeRep(), rewrite)
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

// numericOperators are the binary operators an int literal is rewritten into a double under.
var numericOperators = map[string]struct{}{
	operators.Add:           {},
	operators.Subtract:      {},
	operators.Multiply:      {},
	operators.Divide:        {},
	operators.Less:          {},
	operators.LessEquals:    {},
	operators.Greater:       {},
	operators.GreaterEquals: {},
	operators.Equals:        {},
	operators.NotEquals:     {},
}

// arithmeticOperators are the numericOperators whose result is a number.
var arithmeticOperators = map[string]struct{}{
	operators.Add:      {},
	operators.Subtract: {},
	operators.Multiply: {},
	operators.Divide:   {},
}

// intLiterals adds to rewrite the int literals of a to be rewritten into doubles, which are the
// operands of a numericOperators operator whose other operand does not stay an int. Measured values are float64 while thresholds are mostly
// written as integers, and CEL has no arithmetic between int and double, so `current > prev + 1`
// would not evaluate otherwise. Overloading the arithmetic operators for mixed operands is not
// an option since the standard ones are singleton functions. A literal paired with an int, such
// as the result of size() or an integer variable, stays an int, since the checker has no
// equality between int and double, and so does any literal elsewhere, such as an index or an
// operand of `%`.
func intLiterals(a, typed *ast.AST, rewrite map[int64]struct{}) {
	for _, e := range ast.MatchDescendants(ast.NavigateAST(a), ast.KindMatcher(ast.CallKind)) {
		c := e.AsCall()
		if _, ok := numericOperators[c.FunctionName()]; !ok {
			continue
		}
		args := c.Args()
		if len(args) != 2 {
			continue
		}
		for i, arg := range args {
			if arg.Kind() != ast.LiteralKind {
				continue
			}
			if _, ok := arg.AsLiteral().(types.Int); !ok {
				continue
			}
			if staysInt(args[1-i], typed) {
				continue
			}
			rewrite[arg.ID()] = struct{}{}
		}
	}
}

// literalOnlyIntLiterals adds to rewrite the int literals of the numericOperators operators
// that are made of literals alone.
func literalOnlyIntLiterals(a *ast.AST, rewrite map[int64]struct{}) {
	for _, e := range ast.MatchDescendants(ast.NavigateAST(a), ast.KindMatcher(ast.CallKind)) {
		if !literalOnly(e) {
			continue
		}
		for _, l := range ast.MatchDescendants(e, ast.ConstantValueMatcher()) {
			if _, ok := l.AsLiteral().(types.Int); ok {
				rewrite[l.ID()] = struct{}{}
			}
		}
	}
}

func literalOnly(e ast.Expr) bool {
	switch e.Kind() {
	case ast.LiteralKind:
		return true
	case ast.CallKind:
		c := e.AsCall()
		if _, ok := numericOperators[c.FunctionName()]; !ok {
			return false
		}
		for _, arg := range c.Args() {
			if !literalOnly(arg) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// staysInt reports whether e is an int after the rewrite. The type the checker gives an
// arithmetic operator is not enough on its own, since `dyn + 1` is typed as an int and becomes a
// double once its literal is rewritten, so an arithmetic operator stays an int only when both of
// its operands do.
func staysInt(e ast.Expr, typed *ast.AST) bool {
	switch e.Kind() {
	case ast.LiteralKind:
		_, ok := e.AsLiteral().(types.Int)
		return ok
	case ast.CallKind:
		c := e.AsCall()
		if _, ok := arithmeticOperators[c.FunctionName()]; ok {
			for _, arg := range c.Args() {
				if !staysInt(arg, typed) {
					return false
				}
			}
			return true
		}
	}
	return typed.GetType(e.ID()).Kind() == types.IntKind
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
