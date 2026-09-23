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

// Compile compiles a condition that refers to the variables named names.
func Compile(src string, names []string) (*Program, error) {
	p := &Program{src: src}
	prg, err := compileCEL(src, names)
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
	names := make([]string, 0, len(vars))
	for k := range vars {
		names = append(names, k)
	}
	p, err := Compile(src, names)
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

func compileCEL(src string, names []string) (cel.Program, error) {
	opts := []cel.EnvOption{
		cel.CrossTypeNumericComparisons(true),
		cel.OptionalTypes(),
	}
	for _, n := range names {
		opts = append(opts, cel.Variable(n, cel.DynType))
	}
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, err
	}
	a, iss := env.Parse(src)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	intLiteralsToDouble(a.NativeRep())
	checked, iss := env.Check(a)
	if iss.Err() != nil {
		// The literals compared with an int that is typed statically, such as the result of
		// size(), have to stay ints, since the checker has no equality between int and double.
		a, _ = env.Parse(src)
		var iss2 *cel.Issues
		checked, iss2 = env.Check(a)
		if iss2.Err() != nil {
			return nil, iss.Err()
		}
	}
	return env.Program(checked)
}

// intLiteralsToDouble rewrites the int literals into double ones. Measured values are float64
// while thresholds are mostly written as integers, and CEL has no arithmetic between int and
// double, so `current > prev + 1` would not evaluate otherwise. Overloading the arithmetic
// operators for mixed operands is not an option since the standard ones are singleton
// functions. The operands of an index and of `%` are left as they are, as those take ints.
func intLiteralsToDouble(a *ast.AST) {
	root := ast.NavigateAST(a)
	keep := map[int64]struct{}{}
	for _, e := range ast.MatchDescendants(root, ast.KindMatcher(ast.CallKind)) {
		c := e.AsCall()
		switch c.FunctionName() {
		case operators.Index, operators.OptIndex:
			keep[c.Args()[1].ID()] = struct{}{}
		case operators.Modulo:
			for _, arg := range c.Args() {
				keep[arg.ID()] = struct{}{}
			}
		}
	}
	fac := ast.NewExprFactory()
	for _, e := range ast.MatchDescendants(root, ast.ConstantValueMatcher()) {
		if _, ok := keep[e.ID()]; ok {
			continue
		}
		if i, ok := e.AsLiteral().(types.Int); ok {
			e.SetKindCase(fac.NewLiteral(e.ID(), types.Double(i)))
		}
	}
}

// celVars converts the integer variables into float64 ones, as the literals they are compared
// with have been turned into doubles by intLiteralsToDouble.
func celVars(vars map[string]any) map[string]any {
	converted := make(map[string]any, len(vars))
	for k, v := range vars {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			converted[k] = float64(rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			converted[k] = float64(rv.Uint())
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
