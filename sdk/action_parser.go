package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk/parser"
)

var extractRegexp = regexp.MustCompile("(\\${{.+?}})")

type ActionParser struct {
	contexts map[string]interface{}
	funcs    map[string]ActionFunc
}

func NewActionParser(contexts map[string]interface{}, funcs map[string]ActionFunc) *ActionParser {
	return &ActionParser{
		contexts: contexts,
		funcs:    funcs,
	}
}

func (a *ActionParser) createLexerAndParser(input string) (*parser.ActionParser, *ParserErrorListener) {
	lexer := parser.NewActionLexer(antlr.NewInputStream(input))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewActionParser(stream)

	errorLst := NewParserErrorListener()
	p.RemoveErrorListeners()
	p.AddErrorListener(errorLst)
	p.BuildParseTrees = true
	return p, errorLst
}

func (a *ActionParser) Validate(ctx context.Context, input string) error {
	matches := extractRegexp.FindAllString(input, -1)
	validationInputs := make([]string, 0)
	for _, match := range matches {
		log.Debug(ctx, "Parse expression: %s", match)
		errors := a.checkSyntax(ctx, match)
		validationInputs = append(validationInputs, errors...)
	}

	if len(validationInputs) > 0 {
		return NewErrorFrom(ErrInvalidData, strings.Join(validationInputs, "\n"))
	}
	return nil
}

func (a *ActionParser) checkSyntax(_ context.Context, input string) []string {
	p, errorLst := a.createLexerAndParser(input)
	_ = p.Expression()
	return errorLst.Errors
}

func (a *ActionParser) InterpolateToBool(ctx context.Context, input string) (bool, error) {
	resultString, err := a.InterpolateToString(ctx, input)
	if err != nil {
		return false, err
	}
	result, err := strconv.ParseBool(resultString)
	if err != nil {
		return false, fmt.Errorf("unable to interpolate [%s] into a boolean, got %s: %v", input, resultString, err)
	}
	return result, nil
}

func (a *ActionParser) InterpolateToString(ctx context.Context, input string) (string, error) {
	resultInterface, err := a.Interpolate(ctx, input)
	if err != nil {
		return "", err
	}
	result, ok := resultInterface.(string)
	if !ok {
		return "", fmt.Errorf("unable to interpolate [%s] into a string: %v", input, err)
	}
	return result, nil
}

func (a *ActionParser) Interpolate(ctx context.Context, input string) (interface{}, error) {
	interpolatedInput := input
	matches := extractRegexp.FindAllString(input, -1)
	for _, match := range matches {
		log.Debug(ctx, "Parse expression: %s", match)
		result, err := a.parse(ctx, match)
		if err != nil {
			return input, err
		}
		switch result.(type) {
		case map[string]interface{}, []interface{}:
			if len(matches) == 1 {
				return result, nil
			}
			bts, err := json.Marshal(result)
			if err != nil {
				return nil, NewErrorFrom(ErrInvalidData, "unable to stringify %s: %v", match, err)
			}
			interpolatedInput = strings.Replace(interpolatedInput, match, fmt.Sprintf("%s", string(bts)), 1)
		default:
			interpolatedInput = strings.Replace(interpolatedInput, match, fmt.Sprintf("%v", result), 1)
		}

	}
	return interpolatedInput, nil
}

func (a *ActionParser) parse(ctx context.Context, elt string) (interface{}, error) {
	p, errorLst := a.createLexerAndParser(elt)
	tree := p.Expression()
	if len(errorLst.Errors) != 0 {
		return "", fmt.Errorf(strings.Join(errorLst.Errors, "\n"))
	}

	for i := 0; i < tree.GetChildCount(); i++ {
		switch t := tree.GetChild(i).(type) {
		case *parser.OrExpressionContext:
			return a.parseOrExpressionContext(ctx, t)
		case *parser.ExpressionStartContext, *parser.ExpressionEndContext:
			continue
		default:
			log.Error(ctx, "Unknown child type %T: %s", t, tree.GetText())
		}
	}
	return "", NewErrorFrom(ErrInvalidData, "No expression found. Gor [%s]", elt)
}

func (a *ActionParser) parseOrExpressionContext(ctx context.Context, exp *parser.OrExpressionContext) (interface{}, error) {
	log.Debug(ctx, "Or expression detected: %s", exp.GetText())
	operands := make([]interface{}, 0, exp.GetChildCount())
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *parser.AndExpressionContext:
			result, err := a.parseAndExpressionContext(ctx, t)
			if err != nil {
				return "", err
			}
			operands = append(operands, result)
		case *parser.OrOperatorContext:
			continue
		default:
			return "", NewErrorFrom(ErrInvalidData, "unknown type %T in OrExpression", t)
		}
	}
	switch len(operands) {
	case 1:
		return operands[0], nil
	default:
		b, err := a.or(operands)
		if err != nil {
			return "", err
		}
		return b, nil
	}
}

func (a *ActionParser) parseAndExpressionContext(ctx context.Context, exp *parser.AndExpressionContext) (interface{}, error) {
	log.Debug(ctx, "And expression detected: %s", exp.GetText())
	operands := make([]interface{}, 0, exp.GetChildCount())
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *parser.ComparisonExpressionContext:
			result, err := a.parseComparisonExpressionContext(ctx, t)
			if err != nil {
				return "", err
			}
			operands = append(operands, result)
		case *parser.AndOperatorContext:
			continue
		default:
			return "", NewErrorFrom(ErrInvalidData, "unknown type %T in AndExpression", t)
		}
	}

	switch len(operands) {
	case 1:
		return operands[0], nil
	default:
		return a.and(operands)
	}
}

func (a *ActionParser) parseComparisonExpressionContext(ctx context.Context, exp *parser.ComparisonExpressionContext) (interface{}, error) {
	log.Debug(ctx, "Comparison expression detected: %s", exp.GetText())
	operands := make([]interface{}, 0, exp.GetChildCount())
	var operator string
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *parser.EqualityExpressionContext:
			result, err := a.parseEqualityExpressionContext(ctx, t)
			if err != nil {
				return "", err
			}
			operands = append(operands, result)
		case *parser.ComparisonOperatorContext:
			operator = t.GetText()
			continue
		default:
			return "", NewErrorFrom(ErrInvalidData, "unknown type %T in ComparisonExpression", t)
		}
	}

	switch len(operands) {
	case 1:
		return operands[0], nil
	case 2:
		return a.compare(operands, operator)
	}
	return nil, NewErrorFrom(ErrInvalidData, "invalid comparison expression: %s", exp.GetText())
}

func (a *ActionParser) parseEqualityExpressionContext(ctx context.Context, exp *parser.EqualityExpressionContext) (interface{}, error) {
	log.Debug(ctx, "Equality expression detected: %s", exp.GetText())
	operands := make([]interface{}, 0, exp.GetChildCount())
	var operator string
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *parser.PrimaryExpressionContext:
			result, err := a.parsePrimaryExpressionContext(ctx, t)
			if err != nil {
				return "", err
			}
			operands = append(operands, result)
		case *parser.EqualityOperatorContext:
			operator = t.GetText()
		default:
			return "", NewErrorFrom(ErrInvalidData, "unknown type %T in EqualityExpression", t)
		}
	}

	switch len(operands) {
	case 1:
		return operands[0], nil
	case 2:
		return a.equal(operands, operator)
	}
	return "", NewErrorFrom(ErrInvalidData, "wrong equality expression. Got [%s]", exp.GetText())
}

func (a *ActionParser) parsePrimaryExpressionContext(ctx context.Context, exp *parser.PrimaryExpressionContext) (interface{}, error) {
	log.Debug(ctx, "Primary expression detected: %s", exp.GetText())
	if exp.GetChildCount() != 1 {
		return nil, NewErrorFrom(ErrInvalidData, "invalid empty primary expression. Got [%s]", exp.GetText())
	}

	switch t := exp.GetChild(0).(type) {
	case *parser.VariableContextContext:
		return a.parseVariableContextContext(ctx, t)
	case *parser.FunctionCallContext:
		return a.parseFunctionCallContext(ctx, t)
	case *parser.NumberExpressionContext:
		return a.parseNumberExpression(ctx, t)
	case *parser.StringExpressionContext:
		return a.trimString(t.GetText()), nil
	case *parser.TermExpressionContext:
		return a.parseTermExpressionContext(ctx, t)
	case *parser.NotExpressionContext:
		return a.parseNotExpression(ctx, t)
	default:
		return nil, NewErrorFrom(ErrInvalidData, "unknown type %T in PrimaryExpression", t)
	}
}

func (a *ActionParser) parseNotExpression(ctx context.Context, exp *parser.NotExpressionContext) (bool, error) {
	log.Debug(ctx, "Not expression detected: %s", exp.GetText())
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *parser.NotOperatorContext:
			continue
		case *parser.PrimaryExpressionContext:
			result, err := a.parsePrimaryExpressionContext(ctx, t)
			if err != nil {
				return false, err
			}
			resultB, ok := result.(bool)
			if !ok {
				return false, NewErrorFrom(ErrInvalidData, "expression [%s] need to return a boolean value to be able to use the Not operator. Got [%v]", t.GetText(), result)
			}
			return !resultB, nil
		}
	}
	return false, nil
}

func (a *ActionParser) parseFunctionCallContext(ctx context.Context, exp *parser.FunctionCallContext) (interface{}, error) {
	log.Debug(ctx, "Function call expression detected: %s", exp.GetText())
	var funcName string
	args := make([]interface{}, 0)
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *parser.FunctionNameContext:
			funcName = t.GetText()
		case *parser.FunctionCallArgumentsContext:
			result, err := a.parseFunctionCallArgumentsContext(ctx, t)
			if err != nil {
				return nil, err
			}
			if result == nil {
				continue
			}
			args = append(args, result)
		case *antlr.TerminalNodeImpl:
			if t.GetText() != "(" && t.GetText() != ")" && t.GetText() != "," {
				return nil, NewErrorFrom(ErrInvalidData, "unknown string found in functionCall expression: [%s]", t.GetText())
			}
		default:
			return nil, NewErrorFrom(ErrInvalidData, "unknown type %T in FunctionCall", t)
		}
	}
	return a.callFunction(ctx, funcName, args)
}

func (a *ActionParser) parseFunctionCallArgumentsContext(ctx context.Context, exp *parser.FunctionCallArgumentsContext) (interface{}, error) {
	log.Debug(ctx, "Function call arguments expression detected: %s", exp.GetText())
	if exp.GetChildCount() == 0 {
		return nil, nil
	}

	switch t := exp.GetChild(0).(type) {
	case *parser.VariableContextContext:
		result, err := a.parseVariableContextContext(ctx, t)
		if err != nil {
			return nil, err
		}
		return result, nil
	case *parser.StringExpressionContext:
		return a.trimString(t.GetText()), nil
	case *parser.NumberExpressionContext:
		return a.parseNumberExpression(ctx, t)
	case *parser.BooleanExpressionContext:
		s := t.GetText()
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to parse boolean %s", s)
		}
		return b, nil
	default:
		return nil, NewErrorFrom(ErrInvalidData, "unknown type %T in FunctionCall Arguments", t)
	}

}

func (a *ActionParser) parseNumberExpression(ctx context.Context, t *parser.NumberExpressionContext) (interface{}, error) {
	log.Debug(ctx, "Number expression detected: %s", t.GetText())
	intValue, err := strconv.Atoi(t.GetText())
	if err == nil {
		return intValue, nil
	}
	floatValue, err := strconv.ParseFloat(t.GetText(), 64)
	if err != nil {
		return nil, NewErrorFrom(ErrInvalidData, "value %s is not an int neither a float", t.GetText())
	}
	return floatValue, nil
}

func (a *ActionParser) parseTermExpressionContext(ctx context.Context, exp *parser.TermExpressionContext) (bool, error) {
	log.Debug(ctx, "Term expression detected: %s", exp.GetText())
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *antlr.TerminalNodeImpl:
		case *parser.OrExpressionContext:
			result, err := a.parseOrExpressionContext(ctx, t)
			if err != nil {
				return false, err
			}
			b, err := strconv.ParseBool(fmt.Sprintf("%v", result))
			if err != nil {
				return false, NewErrorFrom(ErrInvalidData, "%s is not a boolean", result)
			}
			return b, nil
		default:
			return false, NewErrorFrom(ErrInvalidData, "unknown type %T in TermExpression", t)
		}
	}
	return false, nil
}

func (a *ActionParser) parseVariableContextContext(ctx context.Context, exp *parser.VariableContextContext) (interface{}, error) {
	log.Debug(ctx, "VariableContext expression detected: %s", exp.GetText())
	var selectedValue interface{}
	isFilter := false
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *parser.VariableIdentifierContext:
			contextName := t.GetText()
			var has bool
			selectedValue, has = a.contexts[contextName]
			if !has {
				return nil, NewErrorFrom(ErrInvalidData, "unknown context %s", contextName)
			}
			continue
		case *parser.VariablePathContext:
			key, err := a.parseVariablePathContext(ctx, t)
			if err != nil {
				return nil, err
			}
			switch kType := key.(type) {
			case string:
				if key == "*" {
					if isFilter {
						return nil, NewErrorFrom(ErrInvalidData, "unable to filter a filtered object")
					}
					isFilter = true
				} else {
					selectedValue, err = a.getItemValueFromContext(ctx, selectedValue, kType, isFilter)
					if err != nil {
						return nil, err
					}
				}
			case int:
				selectedValue, err = a.getArrayItemValueFromContext(ctx, selectedValue, kType)
				if err != nil {
					return nil, err
				}
			}
		default:
			return false, NewErrorFrom(ErrInvalidData, "unknown type %T in VariableContext", t)
		}
	}
	return selectedValue, nil
}

func (a *ActionParser) parseVariablePathContext(ctx context.Context, exp *parser.VariablePathContext) (interface{}, error) {
	log.Debug(ctx, "VariablePath expression detected: %s", exp.GetText())
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *antlr.TerminalNodeImpl:
			continue
		case *parser.VariableIdentifierContext:
			return t.GetText(), nil
		case *parser.ArrayContext:
			result, err := a.parseArrayContext(ctx, t)
			if err != nil {
				return "", err
			}
			return result, nil
		case *parser.FilterExpressionContext:
			log.Debug(ctx, "Filter expression detected: %s", t.GetText())
			return "*", nil
		default:
			return "", NewErrorFrom(ErrInvalidData, "unknown type %T in VariablePath", t)
		}
	}
	return "", nil
}

func (a *ActionParser) parseArrayContext(ctx context.Context, exp *parser.ArrayContext) (int, error) {
	log.Debug(ctx, "ArrayContext expression detected: %s", exp.GetText())
	for i := 0; i < exp.GetChildCount(); i++ {
		switch t := exp.GetChild(i).(type) {
		case *antlr.TerminalNodeImpl:
			if t.GetText() == "[" || t.GetText() == "]" {
				continue
			}
			return 0, NewErrorFrom(ErrInvalidData, "unknown string found in a array context expression: [%s]", t.GetText())
		case *parser.ArrayIndexContext:
			return a.parseArrayIndexContext(ctx, t)
		default:
			return 0, NewErrorFrom(ErrInvalidData, "unknown type %T in ArrayContext", t)
		}
	}
	return 0, NewErrorFrom(ErrInvalidData, "array index not found in expression [%s]", exp.GetText())
}

func (a *ActionParser) parseArrayIndexContext(ctx context.Context, exp *parser.ArrayIndexContext) (int, error) {
	log.Debug(ctx, "ArrayIndexContext expression detected: %s", exp.GetText())
	if exp.GetChildCount() != 1 {
		return 0, NewErrorFrom(ErrInvalidData, "invalid array index expression: [%s]", exp.GetText())
	}
	switch t := exp.GetChild(0).(type) {
	case *parser.PrimaryExpressionContext:
		result, err := a.parsePrimaryExpressionContext(ctx, t)
		if err != nil {
			return 0, err
		}
		resultInt, ok := result.(int)
		if !ok {
			return 0, NewErrorFrom(ErrInvalidData, "invalid int index [%v]", result)
		}
		return resultInt, nil
	default:
		return 0, NewErrorFrom(ErrInvalidData, "invalid index expression %T", t)
	}
}

func (a *ActionParser) getArrayItemValueFromContext(_ context.Context, currentContext interface{}, key int) (interface{}, error) {
	items := reflect.ValueOf(currentContext)
	if items.Kind() != reflect.Slice {
		return nil, NewErrorFrom(ErrInvalidData, "object is not an array. Got %T [%v]", currentContext, currentContext)
	}
	return items.Index(key).Interface(), nil
}

// Item not found in a context must return empty
func (a *ActionParser) getItemValueFromContext(ctx context.Context, currentContext interface{}, key string, isFilter bool) (interface{}, error) {
	if isFilter {
		varContext, ok := currentContext.([]map[string]interface{})
		if !ok {
			return nil, NewErrorFrom(ErrInvalidData, "unable to filter a non array object")
		}
		result := make([]interface{}, 0, len(varContext))
		for _, currentItem := range varContext {
			result = append(result, currentItem[key])
		}

		return result, nil
	}
	varContext, ok := currentContext.(map[string]interface{})
	if !ok {
		log.Debug(ctx, "key [%s] do not exist in current context")
		return "", nil
	}
	value, has := varContext[key]
	if !has {
		return "", nil
	}
	return value, nil
}

func (a *ActionParser) compare(operands []interface{}, operator string) (bool, error) {
	if len(operands) != 2 {
		return false, NewErrorFrom(ErrInvalidData, "cannot compare more or less than 2 operands")
	}
	operand1, ok := operands[0].(float64)
	if !ok {
		operand1Int, ok := operands[0].(int)
		if !ok {
			return false, NewErrorFrom(ErrInvalidData, "%v must be a float or an int, got [%T]", operands[0], operands[0])
		}
		operand1 = float64(operand1Int)
	}
	operand2, ok := operands[1].(float64)
	if !ok {
		operand2Int, ok := operands[1].(int)
		if !ok {
			return false, NewErrorFrom(ErrInvalidData, "%v must be a float or an int, got [%T]", operands[1], operands[1])
		}
		operand2 = float64(operand2Int)
	}

	switch operator {
	case "<":
		return operand1 < operand2, nil
	case ">":
		return operand1 > operand2, nil
	case "<=":
		return operand1 <= operand2, nil
	case ">=":
		return operand1 >= operand2, nil
	default:
		return false, NewErrorFrom(ErrInvalidData, "unknown comparator operator %s", operator)
	}
}

func (a *ActionParser) equal(operands []interface{}, operator string) (bool, error) {
	if len(operands) != 2 {
		return false, NewErrorFrom(ErrInvalidData, "cannot equalize more or less than 2 operands")
	}

	switch operator {
	case "==":
		return operands[0] == operands[1], nil
	case "!=":
		return operands[0] != operands[1], nil
	default:
		return false, NewErrorFrom(ErrInvalidData, "unknown equalizer operator %s", operator)
	}
}

func (a *ActionParser) and(operands []interface{}) (bool, error) {
	if len(operands) == 0 {
		return false, nil
	}
	if len(operands) == 1 {
		return false, NewErrorFrom(ErrInvalidData, "and expression must have at least 2 operands")
	}
	currentResult := true
	for _, op := range operands {
		opBool, ok := op.(bool)
		if !ok {
			return false, NewErrorFrom(ErrInvalidData, "operator && only work with boolean, got [%v]", op)
		}
		currentResult = currentResult && opBool
		if !currentResult {
			return currentResult, nil
		}
	}
	return currentResult, nil
}

func (a *ActionParser) or(operands []interface{}) (bool, error) {
	if len(operands) == 0 {
		return false, nil
	}
	if len(operands) == 1 {
		return false, NewErrorFrom(ErrInvalidData, "and expression must have at least 2 operands")
	}
	currentResult := false
	for _, op := range operands {
		opBool, ok := op.(bool)
		if !ok {
			return false, NewErrorFrom(ErrInvalidData, "operator && only work with boolean, got [%v]", opBool)
		}
		currentResult = currentResult || opBool
		if currentResult {
			return currentResult, nil
		}
	}
	return currentResult, nil
}

func (a *ActionParser) trimString(value string) string {
	return strings.TrimPrefix(strings.TrimSuffix(value, "'"), "'")
}

func (a *ActionParser) callFunction(ctx context.Context, funcName string, args []interface{}) (interface{}, error) {
	fct, has := a.funcs[funcName]
	if !has {
		return nil, NewErrorFrom(ErrInvalidData, "function %s does not exist", funcName)
	}
	return fct(ctx, a, args...)
}
