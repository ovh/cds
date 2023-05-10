package sdk

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
)

type ParserErrorListener struct {
	*antlr.DiagnosticErrorListener
	Errors []string
}

func NewParserErrorListener() *ParserErrorListener {
	del := new(antlr.DiagnosticErrorListener)
	p := &ParserErrorListener{}
	p.DiagnosticErrorListener = del
	return p
}

func (c *ParserErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	var errorMessage string
	a, ok := recognizer.(*antlr.BaseParser)
	if ok {
		errorMessage += fmt.Sprintf("%v: ", a.Consume().GetInputStream())
	}
	c.Errors = append(c.Errors, errorMessage+msg)
}
