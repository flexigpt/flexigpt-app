package workflowv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const listStr = "list"

type expressionScope struct {
	steps         map[string]bool
	reserved      map[string]bool
	variables     map[string]bool
	subscriptions map[string]Subscription
	state         bool
	repeat        bool
	path          string
	depth         int
}

func rootExpressionScope() expressionScope {
	return expressionScope{
		steps:         map[string]bool{},
		reserved:      map[string]bool{},
		variables:     map[string]bool{"input": true, "context": true, "messages": true},
		subscriptions: map[string]Subscription{},
	}
}

func (s expressionScope) clone() expressionScope {
	s.steps = maps.Clone(s.steps)
	s.reserved = maps.Clone(s.reserved)
	s.variables = maps.Clone(s.variables)
	s.subscriptions = maps.Clone(s.subscriptions)
	return s
}

func (s expressionScope) with(name string) expressionScope {
	s = s.clone()
	s.variables[name] = true
	return s
}

func (s expressionScope) child(segment string) expressionScope {
	s = s.clone()
	if s.path != "" {
		s.path += "/"
	}
	s.path += segment
	s.depth++
	return s
}

func (s expressionScope) validate(value Value, expected string) error {
	if len(value) == 0 {
		return nil
	}
	decoded, err := decodeValueJSON(value)
	if err != nil {
		return err
	}
	return s.validateDecoded(decoded, expected)
}

func decodeValueJSON(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, invalid("Workflow value contains multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

func (s expressionScope) validateDecoded(value any, expected string) error {
	switch typed := value.(type) {
	case map[string]any:
		if literal, found := typed["$literal"]; found {
			if len(typed) != 1 {
				return invalid("a $literal directive must have exactly one field")
			}
			return validateLiteralType(literal, expected)
		}
		if raw, found := typed["$expr"]; found {
			source, ok := raw.(string)
			if !ok || len(typed) != 1 || source == "" || len(source) > 65536 {
				return invalid("an expression must be a bounded single-key $expr object")
			}
			return s.compile(source, expected)
		}
		if expected != "" {
			return invalid("expected %s or a CEL expression", expected)
		}
		for _, key := range sortedKeys(typed) {
			if err := s.validateDecoded(typed[key], ""); err != nil {
				return fmt.Errorf("value field %q: %w", key, err)
			}
		}
	case []any:
		if expected != "" && expected != listStr {
			return invalid("expected %s, not a list", expected)
		}
		for index, child := range typed {
			if err := s.validateDecoded(child, ""); err != nil {
				return fmt.Errorf("value item %d: %w", index, err)
			}
		}
	default:
		return validateLiteralType(value, expected)
	}
	return nil
}

func validateLiteralType(value any, expected string) error {
	switch expected {
	case "":
		return nil
	case "bool":
		if _, ok := value.(bool); ok {
			return nil
		}
	case "string":
		if _, ok := value.(string); ok {
			return nil
		}
	case "int":
		switch number := value.(type) {
		case json.Number:
			rational, ok := new(big.Rat).SetString(number.String())
			if ok && rational.Sign() >= 0 && rational.IsInt() &&
				rational.Num().BitLen() <= 53 {
				return nil
			}
		case float64:
			if number >= 0 && number <= 9007199254740991 &&
				number == float64(int64(number)) {
				return nil
			}
		}
	case listStr:
		if _, ok := value.([]any); ok {
			return nil
		}
	}
	return invalid("literal does not have required type %s", expected)
}

func (s expressionScope) compile(source, expected string) error {
	options := []cel.EnvOption{
		cel.Variable("steps", cel.MapType(cel.StringType, cel.DynType)),
	}
	for _, name := range sortedKeys(s.variables) {
		options = append(options, cel.Variable(name, cel.DynType))
	}
	if s.state {
		options = append(options, cel.Variable("state", cel.DynType))
	}
	env, err := cel.NewEnv(options...)
	if err != nil {
		return fmt.Errorf("construct Workflow CEL environment: %w", err)
	}
	a, issues := env.Compile(source)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("%w: CEL: %w", storeSpec.ErrInvalid, issues.Err())
	}
	if expected != "" {
		actual := a.OutputType().String()
		if actual != expected && actual != "dyn" &&
			(expected != listStr || !strings.HasPrefix(actual, "list(")) {
			return invalid("CEL result type %s does not satisfy %s", actual, expected)
		}
	}
	return s.checkReferences(a.NativeRep().Expr(), map[string]bool{})
}

// checkReferences runs on CEL's checked AST, not on source text. Comprehension
// variables are lexical locals and do not acquire access to Workflow scopes.
func (s expressionScope) checkReferences(expression ast.Expr, bound map[string]bool) error {
	if expression == nil {
		return nil
	}
	if root, fields, dynamic := accessPath(expression); root != "" && !bound[root] {
		switch root {
		case "steps":
			if dynamic && len(fields) == 0 {
				return invalid("the first steps index must be a literal step ID")
			}
			if len(fields) > 0 && !s.steps[fields[0]] {
				return invalid("step %q is unknown, unfinished, or outside this scope", fields[0])
			}
		default:
		}
	}
	switch expression.Kind() {
	case ast.SelectKind:
		return s.checkReferences(expression.AsSelect().Operand(), bound)
	case ast.CallKind:
		call := expression.AsCall()
		if call.IsMemberFunction() {
			if err := s.checkReferences(call.Target(), bound); err != nil {
				return err
			}
		}
		for _, argument := range call.Args() {
			if err := s.checkReferences(argument, bound); err != nil {
				return err
			}
		}
	case ast.ListKind:
		for _, element := range expression.AsList().Elements() {
			if err := s.checkReferences(element, bound); err != nil {
				return err
			}
		}
	case ast.MapKind:
		for _, entry := range expression.AsMap().Entries() {
			value := entry.AsMapEntry()
			if err := s.checkReferences(value.Key(), bound); err != nil {
				return err
			}
			if err := s.checkReferences(value.Value(), bound); err != nil {
				return err
			}
		}
	case ast.StructKind:
		for _, field := range expression.AsStruct().Fields() {
			if err := s.checkReferences(field.AsStructField().Value(), bound); err != nil {
				return err
			}
		}
	case ast.ComprehensionKind:
		value := expression.AsComprehension()
		if err := s.checkReferences(value.IterRange(), bound); err != nil {
			return err
		}
		if err := s.checkReferences(value.AccuInit(), bound); err != nil {
			return err
		}
		locals := maps.Clone(bound)
		locals[value.IterVar()] = true
		locals[value.AccuVar()] = true
		if name := value.IterVar2(); name != "" {
			locals[name] = true
		}
		for _, child := range []ast.Expr{
			value.LoopCondition(),
			value.LoopStep(),
			value.Result(),
		} {
			if err := s.checkReferences(child, locals); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

func accessPath(expression ast.Expr) (r string, p []string, d bool) {
	if expression == nil {
		return "", nil, false
	}
	switch expression.Kind() {
	case ast.IdentKind:
		return expression.AsIdent(), nil, false
	case ast.SelectKind:
		selected := expression.AsSelect()
		root, path, dynamic := accessPath(selected.Operand())
		return root, append(path, selected.FieldName()), dynamic
	case ast.CallKind:
		call := expression.AsCall()
		if call.FunctionName() != "_[_]" || len(call.Args()) != 2 {
			return "", nil, false
		}
		root, path, dynamic := accessPath(call.Args()[0])
		keyExpr := call.Args()[1]
		if keyExpr.Kind() != ast.LiteralKind {
			return root, path, true
		}
		key, ok := keyExpr.AsLiteral().Value().(string)
		if !ok {
			return root, path, true
		}
		return root, append(path, key), dynamic
	default:
		return "", nil, false
	}
}
