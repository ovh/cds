package engine

import (
	"fmt"
	"strconv"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
)

// Operator compares 2 values and return a boolean
type Operator func(leftValue Value, rightValue Value) bool

// NewOperator initializes the operator matching the Token number
func NewOperator(token int, lexeme string) (Operator, error) {
	switch token {
	case parser.EqualityToken:
		return equalityOperator, nil
	case parser.LeftDipleToken:
		return lessThanOperator, nil
	case parser.RightDipleToken:
		return greaterThanOperator, nil
	}

	return nil, fmt.Errorf("Operator '%s' does not exist", lexeme)
}

func convToFloat(t interface{}) (float64, error) {

	switch t := t.(type) {
	default:
		log.Debug("convToFloat> unexpected type %T\n", t)
		return 0, fmt.Errorf("unexpected internal type %T", t)
	case float64:
		return float64(t), nil
	case int64:
		return float64(int64(t)), nil
	case int:
		return float64(int(t)), nil
	case string:
		return strconv.ParseFloat(string(t), 64)
	}

}

func greaterThanOperator(leftValue Value, rightValue Value) bool {
	log.Debug("GreaterThanOperator")
	var left, right float64
	var err error

	var rvalue interface{}
	if rightValue.v != nil {
		rvalue = rightValue.v
	} else {
		rvalue = rightValue.lexeme
	}

	left, err = convToFloat(leftValue.v)
	if err != nil {
		log.Debug("GreateThanOperator> %s\n", err)
		return false
	}

	right, err = convToFloat(rvalue)
	if err != nil {
		log.Debug("GreateThanOperator> %s\n", err)
		return false
	}

	return left > right
}

func lessThanOperator(leftValue Value, rightValue Value) bool {
	log.Debug("LessThanOperator")
	var left, right float64
	var err error

	var rvalue interface{}
	if rightValue.v != nil {
		rvalue = rightValue.v
	} else {
		rvalue = rightValue.lexeme
	}

	left, err = convToFloat(leftValue.v)
	if err != nil {
		log.Debug("lessThanOperator> %s\n", err)
		return false
	}

	right, err = convToFloat(rvalue)
	if err != nil {
		log.Debug("lessThanOperator> %s\n", err)
		return false
	}

	return left < right
}

// EqualityOperator checks if given value are equal
func equalityOperator(leftValue Value, rightValue Value) bool {

	if fmt.Sprintf("%v", leftValue.v) == rightValue.lexeme {
		return true
	}

	return false
}

// TrueOperator always returns true
func TrueOperator(leftValue Value, rightValue Value) bool {
	return true
}

func inOperator(leftValue Value, rightValue Value) bool {
	// Right value should be a slice of string
	values, ok := rightValue.v.([]string)
	if !ok {
		log.Debug("InOperator: rightValue.v is not a []string !")
		return false
	}

	for i := range values {
		log.Debug("InOperator: Testing %v against %s", leftValue.v, values[i])
		if fmt.Sprintf("%v", leftValue.v) == values[i] {
			return true
		}
	}

	return false
}

func isNullOperator(leftValue Value, rightValue Value) bool {
	return leftValue.v == nil
}

func isNotNullOperator(leftValue Value, rightValue Value) bool {
	return leftValue.v != nil
}
