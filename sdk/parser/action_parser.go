// Code generated from Action.g4 by ANTLR 4.12.0. DO NOT EDIT.

package parser // Action

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type ActionParser struct {
	*antlr.BaseParser
}

var actionParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	literalNames           []string
	symbolicNames          []string
	ruleNames              []string
	predictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func actionParserInit() {
	staticData := &actionParserStaticData
	staticData.literalNames = []string{
		"", "','", "'['", "']'", "", "", "'null'", "'${{'", "'}}'", "", "'=='",
		"'!='", "'>'", "'<'", "'>='", "'<='", "", "'('", "')'", "'!'", "'||'",
		"'&&'", "'.'",
	}
	staticData.symbolicNames = []string{
		"", "", "", "", "STRING_INSIDE_EXPRESSION", "BOOLEAN", "NULL", "EXP_START",
		"EXP_END", "NUMBER", "EQ", "NEQ", "GT", "LT", "GTE", "LTE", "ID", "LPAREN",
		"RPAREN", "NOT", "OR", "AND", "DOT", "WS",
	}
	staticData.ruleNames = []string{
		"start", "expression", "orExpression", "andExpression", "comparisonExpression",
		"equalityExpression", "primaryExpression", "variableContext", "variablePath",
		"variableIdentifier", "numberExpression", "stringExpression", "termExpression",
		"notExpression", "functionCall", "functionName", "functionCallArguments",
		"array", "arrayIndex", "andOperator", "orOperator", "comparisonOperator",
		"equalityOperator", "literal", "expressionStart", "expressionEnd",
	}
	staticData.predictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 23, 169, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 1, 0,
		1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 5, 1, 59, 8, 1, 10, 1, 12, 1, 62, 9, 1, 1,
		1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 2, 5, 2, 70, 8, 2, 10, 2, 12, 2, 73, 9, 2,
		1, 3, 1, 3, 1, 3, 1, 3, 5, 3, 79, 8, 3, 10, 3, 12, 3, 82, 9, 3, 1, 4, 1,
		4, 1, 4, 1, 4, 3, 4, 88, 8, 4, 1, 5, 1, 5, 1, 5, 1, 5, 3, 5, 94, 8, 5,
		1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 3, 6, 102, 8, 6, 1, 7, 1, 7, 5, 7,
		106, 8, 7, 10, 7, 12, 7, 109, 9, 7, 1, 8, 1, 8, 1, 8, 3, 8, 114, 8, 8,
		1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 12, 1, 12, 1, 12, 1, 12, 1,
		13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 14, 1, 14, 1, 14, 5, 14, 134, 8, 14,
		10, 14, 12, 14, 137, 9, 14, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1,
		16, 1, 16, 3, 16, 147, 8, 16, 1, 17, 1, 17, 1, 17, 1, 17, 1, 18, 1, 18,
		1, 19, 1, 19, 1, 20, 1, 20, 1, 21, 1, 21, 1, 22, 1, 22, 1, 23, 1, 23, 1,
		24, 1, 24, 1, 25, 1, 25, 1, 25, 0, 0, 26, 0, 2, 4, 6, 8, 10, 12, 14, 16,
		18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 0,
		3, 1, 0, 12, 15, 1, 0, 10, 11, 2, 0, 4, 6, 9, 9, 158, 0, 52, 1, 0, 0, 0,
		2, 55, 1, 0, 0, 0, 4, 65, 1, 0, 0, 0, 6, 74, 1, 0, 0, 0, 8, 83, 1, 0, 0,
		0, 10, 89, 1, 0, 0, 0, 12, 101, 1, 0, 0, 0, 14, 103, 1, 0, 0, 0, 16, 113,
		1, 0, 0, 0, 18, 115, 1, 0, 0, 0, 20, 117, 1, 0, 0, 0, 22, 119, 1, 0, 0,
		0, 24, 121, 1, 0, 0, 0, 26, 125, 1, 0, 0, 0, 28, 128, 1, 0, 0, 0, 30, 140,
		1, 0, 0, 0, 32, 146, 1, 0, 0, 0, 34, 148, 1, 0, 0, 0, 36, 152, 1, 0, 0,
		0, 38, 154, 1, 0, 0, 0, 40, 156, 1, 0, 0, 0, 42, 158, 1, 0, 0, 0, 44, 160,
		1, 0, 0, 0, 46, 162, 1, 0, 0, 0, 48, 164, 1, 0, 0, 0, 50, 166, 1, 0, 0,
		0, 52, 53, 3, 2, 1, 0, 53, 54, 5, 0, 0, 1, 54, 1, 1, 0, 0, 0, 55, 56, 3,
		48, 24, 0, 56, 60, 3, 4, 2, 0, 57, 59, 3, 4, 2, 0, 58, 57, 1, 0, 0, 0,
		59, 62, 1, 0, 0, 0, 60, 58, 1, 0, 0, 0, 60, 61, 1, 0, 0, 0, 61, 63, 1,
		0, 0, 0, 62, 60, 1, 0, 0, 0, 63, 64, 3, 50, 25, 0, 64, 3, 1, 0, 0, 0, 65,
		71, 3, 6, 3, 0, 66, 67, 3, 40, 20, 0, 67, 68, 3, 6, 3, 0, 68, 70, 1, 0,
		0, 0, 69, 66, 1, 0, 0, 0, 70, 73, 1, 0, 0, 0, 71, 69, 1, 0, 0, 0, 71, 72,
		1, 0, 0, 0, 72, 5, 1, 0, 0, 0, 73, 71, 1, 0, 0, 0, 74, 80, 3, 8, 4, 0,
		75, 76, 3, 38, 19, 0, 76, 77, 3, 8, 4, 0, 77, 79, 1, 0, 0, 0, 78, 75, 1,
		0, 0, 0, 79, 82, 1, 0, 0, 0, 80, 78, 1, 0, 0, 0, 80, 81, 1, 0, 0, 0, 81,
		7, 1, 0, 0, 0, 82, 80, 1, 0, 0, 0, 83, 87, 3, 10, 5, 0, 84, 85, 3, 42,
		21, 0, 85, 86, 3, 10, 5, 0, 86, 88, 1, 0, 0, 0, 87, 84, 1, 0, 0, 0, 87,
		88, 1, 0, 0, 0, 88, 9, 1, 0, 0, 0, 89, 93, 3, 12, 6, 0, 90, 91, 3, 44,
		22, 0, 91, 92, 3, 12, 6, 0, 92, 94, 1, 0, 0, 0, 93, 90, 1, 0, 0, 0, 93,
		94, 1, 0, 0, 0, 94, 11, 1, 0, 0, 0, 95, 102, 3, 14, 7, 0, 96, 102, 3, 20,
		10, 0, 97, 102, 3, 28, 14, 0, 98, 102, 3, 22, 11, 0, 99, 102, 3, 24, 12,
		0, 100, 102, 3, 26, 13, 0, 101, 95, 1, 0, 0, 0, 101, 96, 1, 0, 0, 0, 101,
		97, 1, 0, 0, 0, 101, 98, 1, 0, 0, 0, 101, 99, 1, 0, 0, 0, 101, 100, 1,
		0, 0, 0, 102, 13, 1, 0, 0, 0, 103, 107, 3, 18, 9, 0, 104, 106, 3, 16, 8,
		0, 105, 104, 1, 0, 0, 0, 106, 109, 1, 0, 0, 0, 107, 105, 1, 0, 0, 0, 107,
		108, 1, 0, 0, 0, 108, 15, 1, 0, 0, 0, 109, 107, 1, 0, 0, 0, 110, 111, 5,
		22, 0, 0, 111, 114, 3, 18, 9, 0, 112, 114, 3, 34, 17, 0, 113, 110, 1, 0,
		0, 0, 113, 112, 1, 0, 0, 0, 114, 17, 1, 0, 0, 0, 115, 116, 5, 16, 0, 0,
		116, 19, 1, 0, 0, 0, 117, 118, 5, 9, 0, 0, 118, 21, 1, 0, 0, 0, 119, 120,
		5, 4, 0, 0, 120, 23, 1, 0, 0, 0, 121, 122, 5, 17, 0, 0, 122, 123, 3, 4,
		2, 0, 123, 124, 5, 18, 0, 0, 124, 25, 1, 0, 0, 0, 125, 126, 5, 19, 0, 0,
		126, 127, 3, 12, 6, 0, 127, 27, 1, 0, 0, 0, 128, 129, 3, 30, 15, 0, 129,
		130, 5, 17, 0, 0, 130, 135, 3, 32, 16, 0, 131, 132, 5, 1, 0, 0, 132, 134,
		3, 32, 16, 0, 133, 131, 1, 0, 0, 0, 134, 137, 1, 0, 0, 0, 135, 133, 1,
		0, 0, 0, 135, 136, 1, 0, 0, 0, 136, 138, 1, 0, 0, 0, 137, 135, 1, 0, 0,
		0, 138, 139, 5, 18, 0, 0, 139, 29, 1, 0, 0, 0, 140, 141, 5, 16, 0, 0, 141,
		31, 1, 0, 0, 0, 142, 147, 1, 0, 0, 0, 143, 147, 3, 14, 7, 0, 144, 147,
		3, 20, 10, 0, 145, 147, 3, 46, 23, 0, 146, 142, 1, 0, 0, 0, 146, 143, 1,
		0, 0, 0, 146, 144, 1, 0, 0, 0, 146, 145, 1, 0, 0, 0, 147, 33, 1, 0, 0,
		0, 148, 149, 5, 2, 0, 0, 149, 150, 3, 36, 18, 0, 150, 151, 5, 3, 0, 0,
		151, 35, 1, 0, 0, 0, 152, 153, 3, 12, 6, 0, 153, 37, 1, 0, 0, 0, 154, 155,
		5, 21, 0, 0, 155, 39, 1, 0, 0, 0, 156, 157, 5, 20, 0, 0, 157, 41, 1, 0,
		0, 0, 158, 159, 7, 0, 0, 0, 159, 43, 1, 0, 0, 0, 160, 161, 7, 1, 0, 0,
		161, 45, 1, 0, 0, 0, 162, 163, 7, 2, 0, 0, 163, 47, 1, 0, 0, 0, 164, 165,
		5, 7, 0, 0, 165, 49, 1, 0, 0, 0, 166, 167, 5, 8, 0, 0, 167, 51, 1, 0, 0,
		0, 10, 60, 71, 80, 87, 93, 101, 107, 113, 135, 146,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// ActionParserInit initializes any static state used to implement ActionParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewActionParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func ActionParserInit() {
	staticData := &actionParserStaticData
	staticData.once.Do(actionParserInit)
}

// NewActionParser produces a new parser instance for the optional input antlr.TokenStream.
func NewActionParser(input antlr.TokenStream) *ActionParser {
	ActionParserInit()
	this := new(ActionParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &actionParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.predictionContextCache)
	this.RuleNames = staticData.ruleNames
	this.LiteralNames = staticData.literalNames
	this.SymbolicNames = staticData.symbolicNames
	this.GrammarFileName = "Action.g4"

	return this
}

// ActionParser tokens.
const (
	ActionParserEOF                      = antlr.TokenEOF
	ActionParserT__0                     = 1
	ActionParserT__1                     = 2
	ActionParserT__2                     = 3
	ActionParserSTRING_INSIDE_EXPRESSION = 4
	ActionParserBOOLEAN                  = 5
	ActionParserNULL                     = 6
	ActionParserEXP_START                = 7
	ActionParserEXP_END                  = 8
	ActionParserNUMBER                   = 9
	ActionParserEQ                       = 10
	ActionParserNEQ                      = 11
	ActionParserGT                       = 12
	ActionParserLT                       = 13
	ActionParserGTE                      = 14
	ActionParserLTE                      = 15
	ActionParserID                       = 16
	ActionParserLPAREN                   = 17
	ActionParserRPAREN                   = 18
	ActionParserNOT                      = 19
	ActionParserOR                       = 20
	ActionParserAND                      = 21
	ActionParserDOT                      = 22
	ActionParserWS                       = 23
)

// ActionParser rules.
const (
	ActionParserRULE_start                 = 0
	ActionParserRULE_expression            = 1
	ActionParserRULE_orExpression          = 2
	ActionParserRULE_andExpression         = 3
	ActionParserRULE_comparisonExpression  = 4
	ActionParserRULE_equalityExpression    = 5
	ActionParserRULE_primaryExpression     = 6
	ActionParserRULE_variableContext       = 7
	ActionParserRULE_variablePath          = 8
	ActionParserRULE_variableIdentifier    = 9
	ActionParserRULE_numberExpression      = 10
	ActionParserRULE_stringExpression      = 11
	ActionParserRULE_termExpression        = 12
	ActionParserRULE_notExpression         = 13
	ActionParserRULE_functionCall          = 14
	ActionParserRULE_functionName          = 15
	ActionParserRULE_functionCallArguments = 16
	ActionParserRULE_array                 = 17
	ActionParserRULE_arrayIndex            = 18
	ActionParserRULE_andOperator           = 19
	ActionParserRULE_orOperator            = 20
	ActionParserRULE_comparisonOperator    = 21
	ActionParserRULE_equalityOperator      = 22
	ActionParserRULE_literal               = 23
	ActionParserRULE_expressionStart       = 24
	ActionParserRULE_expressionEnd         = 25
)

// IStartContext is an interface to support dynamic dispatch.
type IStartContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Expression() IExpressionContext
	EOF() antlr.TerminalNode

	// IsStartContext differentiates from other interfaces.
	IsStartContext()
}

type StartContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStartContext() *StartContext {
	var p = new(StartContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_start
	return p
}

func (*StartContext) IsStartContext() {}

func NewStartContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StartContext {
	var p = new(StartContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_start

	return p
}

func (s *StartContext) GetParser() antlr.Parser { return s.parser }

func (s *StartContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *StartContext) EOF() antlr.TerminalNode {
	return s.GetToken(ActionParserEOF, 0)
}

func (s *StartContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StartContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StartContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterStart(s)
	}
}

func (s *StartContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitStart(s)
	}
}

func (p *ActionParser) Start() (localctx IStartContext) {
	this := p
	_ = this

	localctx = NewStartContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, ActionParserRULE_start)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(52)
		p.Expression()
	}
	{
		p.SetState(53)
		p.Match(ActionParserEOF)
	}

	return localctx
}

// IExpressionContext is an interface to support dynamic dispatch.
type IExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ExpressionStart() IExpressionStartContext
	AllOrExpression() []IOrExpressionContext
	OrExpression(i int) IOrExpressionContext
	ExpressionEnd() IExpressionEndContext

	// IsExpressionContext differentiates from other interfaces.
	IsExpressionContext()
}

type ExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionContext() *ExpressionContext {
	var p = new(ExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_expression
	return p
}

func (*ExpressionContext) IsExpressionContext() {}

func NewExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionContext {
	var p = new(ExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_expression

	return p
}

func (s *ExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionContext) ExpressionStart() IExpressionStartContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionStartContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionStartContext)
}

func (s *ExpressionContext) AllOrExpression() []IOrExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IOrExpressionContext); ok {
			len++
		}
	}

	tst := make([]IOrExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IOrExpressionContext); ok {
			tst[i] = t.(IOrExpressionContext)
			i++
		}
	}

	return tst
}

func (s *ExpressionContext) OrExpression(i int) IOrExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOrExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOrExpressionContext)
}

func (s *ExpressionContext) ExpressionEnd() IExpressionEndContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionEndContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionEndContext)
}

func (s *ExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterExpression(s)
	}
}

func (s *ExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitExpression(s)
	}
}

func (p *ActionParser) Expression() (localctx IExpressionContext) {
	this := p
	_ = this

	localctx = NewExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, ActionParserRULE_expression)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(55)
		p.ExpressionStart()
	}
	{
		p.SetState(56)
		p.OrExpression()
	}
	p.SetState(60)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&721424) != 0 {
		{
			p.SetState(57)
			p.OrExpression()
		}

		p.SetState(62)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(63)
		p.ExpressionEnd()
	}

	return localctx
}

// IOrExpressionContext is an interface to support dynamic dispatch.
type IOrExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllAndExpression() []IAndExpressionContext
	AndExpression(i int) IAndExpressionContext
	AllOrOperator() []IOrOperatorContext
	OrOperator(i int) IOrOperatorContext

	// IsOrExpressionContext differentiates from other interfaces.
	IsOrExpressionContext()
}

type OrExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOrExpressionContext() *OrExpressionContext {
	var p = new(OrExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_orExpression
	return p
}

func (*OrExpressionContext) IsOrExpressionContext() {}

func NewOrExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OrExpressionContext {
	var p = new(OrExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_orExpression

	return p
}

func (s *OrExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *OrExpressionContext) AllAndExpression() []IAndExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IAndExpressionContext); ok {
			len++
		}
	}

	tst := make([]IAndExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IAndExpressionContext); ok {
			tst[i] = t.(IAndExpressionContext)
			i++
		}
	}

	return tst
}

func (s *OrExpressionContext) AndExpression(i int) IAndExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAndExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAndExpressionContext)
}

func (s *OrExpressionContext) AllOrOperator() []IOrOperatorContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IOrOperatorContext); ok {
			len++
		}
	}

	tst := make([]IOrOperatorContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IOrOperatorContext); ok {
			tst[i] = t.(IOrOperatorContext)
			i++
		}
	}

	return tst
}

func (s *OrExpressionContext) OrOperator(i int) IOrOperatorContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOrOperatorContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOrOperatorContext)
}

func (s *OrExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OrExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OrExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterOrExpression(s)
	}
}

func (s *OrExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitOrExpression(s)
	}
}

func (p *ActionParser) OrExpression() (localctx IOrExpressionContext) {
	this := p
	_ = this

	localctx = NewOrExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, ActionParserRULE_orExpression)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(65)
		p.AndExpression()
	}
	p.SetState(71)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserOR {
		{
			p.SetState(66)
			p.OrOperator()
		}
		{
			p.SetState(67)
			p.AndExpression()
		}

		p.SetState(73)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IAndExpressionContext is an interface to support dynamic dispatch.
type IAndExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllComparisonExpression() []IComparisonExpressionContext
	ComparisonExpression(i int) IComparisonExpressionContext
	AllAndOperator() []IAndOperatorContext
	AndOperator(i int) IAndOperatorContext

	// IsAndExpressionContext differentiates from other interfaces.
	IsAndExpressionContext()
}

type AndExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAndExpressionContext() *AndExpressionContext {
	var p = new(AndExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_andExpression
	return p
}

func (*AndExpressionContext) IsAndExpressionContext() {}

func NewAndExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AndExpressionContext {
	var p = new(AndExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_andExpression

	return p
}

func (s *AndExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *AndExpressionContext) AllComparisonExpression() []IComparisonExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IComparisonExpressionContext); ok {
			len++
		}
	}

	tst := make([]IComparisonExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IComparisonExpressionContext); ok {
			tst[i] = t.(IComparisonExpressionContext)
			i++
		}
	}

	return tst
}

func (s *AndExpressionContext) ComparisonExpression(i int) IComparisonExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IComparisonExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IComparisonExpressionContext)
}

func (s *AndExpressionContext) AllAndOperator() []IAndOperatorContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IAndOperatorContext); ok {
			len++
		}
	}

	tst := make([]IAndOperatorContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IAndOperatorContext); ok {
			tst[i] = t.(IAndOperatorContext)
			i++
		}
	}

	return tst
}

func (s *AndExpressionContext) AndOperator(i int) IAndOperatorContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAndOperatorContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAndOperatorContext)
}

func (s *AndExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AndExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AndExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterAndExpression(s)
	}
}

func (s *AndExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitAndExpression(s)
	}
}

func (p *ActionParser) AndExpression() (localctx IAndExpressionContext) {
	this := p
	_ = this

	localctx = NewAndExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, ActionParserRULE_andExpression)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(74)
		p.ComparisonExpression()
	}
	p.SetState(80)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserAND {
		{
			p.SetState(75)
			p.AndOperator()
		}
		{
			p.SetState(76)
			p.ComparisonExpression()
		}

		p.SetState(82)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IComparisonExpressionContext is an interface to support dynamic dispatch.
type IComparisonExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllEqualityExpression() []IEqualityExpressionContext
	EqualityExpression(i int) IEqualityExpressionContext
	ComparisonOperator() IComparisonOperatorContext

	// IsComparisonExpressionContext differentiates from other interfaces.
	IsComparisonExpressionContext()
}

type ComparisonExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyComparisonExpressionContext() *ComparisonExpressionContext {
	var p = new(ComparisonExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_comparisonExpression
	return p
}

func (*ComparisonExpressionContext) IsComparisonExpressionContext() {}

func NewComparisonExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ComparisonExpressionContext {
	var p = new(ComparisonExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_comparisonExpression

	return p
}

func (s *ComparisonExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *ComparisonExpressionContext) AllEqualityExpression() []IEqualityExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IEqualityExpressionContext); ok {
			len++
		}
	}

	tst := make([]IEqualityExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IEqualityExpressionContext); ok {
			tst[i] = t.(IEqualityExpressionContext)
			i++
		}
	}

	return tst
}

func (s *ComparisonExpressionContext) EqualityExpression(i int) IEqualityExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEqualityExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEqualityExpressionContext)
}

func (s *ComparisonExpressionContext) ComparisonOperator() IComparisonOperatorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IComparisonOperatorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IComparisonOperatorContext)
}

func (s *ComparisonExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ComparisonExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ComparisonExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterComparisonExpression(s)
	}
}

func (s *ComparisonExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitComparisonExpression(s)
	}
}

func (p *ActionParser) ComparisonExpression() (localctx IComparisonExpressionContext) {
	this := p
	_ = this

	localctx = NewComparisonExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, ActionParserRULE_comparisonExpression)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(83)
		p.EqualityExpression()
	}
	p.SetState(87)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&61440) != 0 {
		{
			p.SetState(84)
			p.ComparisonOperator()
		}
		{
			p.SetState(85)
			p.EqualityExpression()
		}

	}

	return localctx
}

// IEqualityExpressionContext is an interface to support dynamic dispatch.
type IEqualityExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllPrimaryExpression() []IPrimaryExpressionContext
	PrimaryExpression(i int) IPrimaryExpressionContext
	EqualityOperator() IEqualityOperatorContext

	// IsEqualityExpressionContext differentiates from other interfaces.
	IsEqualityExpressionContext()
}

type EqualityExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEqualityExpressionContext() *EqualityExpressionContext {
	var p = new(EqualityExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_equalityExpression
	return p
}

func (*EqualityExpressionContext) IsEqualityExpressionContext() {}

func NewEqualityExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EqualityExpressionContext {
	var p = new(EqualityExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_equalityExpression

	return p
}

func (s *EqualityExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *EqualityExpressionContext) AllPrimaryExpression() []IPrimaryExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IPrimaryExpressionContext); ok {
			len++
		}
	}

	tst := make([]IPrimaryExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IPrimaryExpressionContext); ok {
			tst[i] = t.(IPrimaryExpressionContext)
			i++
		}
	}

	return tst
}

func (s *EqualityExpressionContext) PrimaryExpression(i int) IPrimaryExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimaryExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimaryExpressionContext)
}

func (s *EqualityExpressionContext) EqualityOperator() IEqualityOperatorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEqualityOperatorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEqualityOperatorContext)
}

func (s *EqualityExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EqualityExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EqualityExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterEqualityExpression(s)
	}
}

func (s *EqualityExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitEqualityExpression(s)
	}
}

func (p *ActionParser) EqualityExpression() (localctx IEqualityExpressionContext) {
	this := p
	_ = this

	localctx = NewEqualityExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, ActionParserRULE_equalityExpression)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(89)
		p.PrimaryExpression()
	}
	p.SetState(93)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == ActionParserEQ || _la == ActionParserNEQ {
		{
			p.SetState(90)
			p.EqualityOperator()
		}
		{
			p.SetState(91)
			p.PrimaryExpression()
		}

	}

	return localctx
}

// IPrimaryExpressionContext is an interface to support dynamic dispatch.
type IPrimaryExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VariableContext() IVariableContextContext
	NumberExpression() INumberExpressionContext
	FunctionCall() IFunctionCallContext
	StringExpression() IStringExpressionContext
	TermExpression() ITermExpressionContext
	NotExpression() INotExpressionContext

	// IsPrimaryExpressionContext differentiates from other interfaces.
	IsPrimaryExpressionContext()
}

type PrimaryExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPrimaryExpressionContext() *PrimaryExpressionContext {
	var p = new(PrimaryExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_primaryExpression
	return p
}

func (*PrimaryExpressionContext) IsPrimaryExpressionContext() {}

func NewPrimaryExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PrimaryExpressionContext {
	var p = new(PrimaryExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_primaryExpression

	return p
}

func (s *PrimaryExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *PrimaryExpressionContext) VariableContext() IVariableContextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableContextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableContextContext)
}

func (s *PrimaryExpressionContext) NumberExpression() INumberExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INumberExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INumberExpressionContext)
}

func (s *PrimaryExpressionContext) FunctionCall() IFunctionCallContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionCallContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionCallContext)
}

func (s *PrimaryExpressionContext) StringExpression() IStringExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStringExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStringExpressionContext)
}

func (s *PrimaryExpressionContext) TermExpression() ITermExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITermExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITermExpressionContext)
}

func (s *PrimaryExpressionContext) NotExpression() INotExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INotExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INotExpressionContext)
}

func (s *PrimaryExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PrimaryExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PrimaryExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterPrimaryExpression(s)
	}
}

func (s *PrimaryExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitPrimaryExpression(s)
	}
}

func (p *ActionParser) PrimaryExpression() (localctx IPrimaryExpressionContext) {
	this := p
	_ = this

	localctx = NewPrimaryExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, ActionParserRULE_primaryExpression)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.SetState(101)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 5, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(95)
			p.VariableContext()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(96)
			p.NumberExpression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(97)
			p.FunctionCall()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(98)
			p.StringExpression()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(99)
			p.TermExpression()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(100)
			p.NotExpression()
		}

	}

	return localctx
}

// IVariableContextContext is an interface to support dynamic dispatch.
type IVariableContextContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VariableIdentifier() IVariableIdentifierContext
	AllVariablePath() []IVariablePathContext
	VariablePath(i int) IVariablePathContext

	// IsVariableContextContext differentiates from other interfaces.
	IsVariableContextContext()
}

type VariableContextContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableContextContext() *VariableContextContext {
	var p = new(VariableContextContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_variableContext
	return p
}

func (*VariableContextContext) IsVariableContextContext() {}

func NewVariableContextContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableContextContext {
	var p = new(VariableContextContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_variableContext

	return p
}

func (s *VariableContextContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableContextContext) VariableIdentifier() IVariableIdentifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableIdentifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableIdentifierContext)
}

func (s *VariableContextContext) AllVariablePath() []IVariablePathContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IVariablePathContext); ok {
			len++
		}
	}

	tst := make([]IVariablePathContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IVariablePathContext); ok {
			tst[i] = t.(IVariablePathContext)
			i++
		}
	}

	return tst
}

func (s *VariableContextContext) VariablePath(i int) IVariablePathContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariablePathContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariablePathContext)
}

func (s *VariableContextContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableContextContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableContextContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterVariableContext(s)
	}
}

func (s *VariableContextContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitVariableContext(s)
	}
}

func (p *ActionParser) VariableContext() (localctx IVariableContextContext) {
	this := p
	_ = this

	localctx = NewVariableContextContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, ActionParserRULE_variableContext)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(103)
		p.VariableIdentifier()
	}
	p.SetState(107)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserT__1 || _la == ActionParserDOT {
		{
			p.SetState(104)
			p.VariablePath()
		}

		p.SetState(109)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IVariablePathContext is an interface to support dynamic dispatch.
type IVariablePathContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	DOT() antlr.TerminalNode
	VariableIdentifier() IVariableIdentifierContext
	Array() IArrayContext

	// IsVariablePathContext differentiates from other interfaces.
	IsVariablePathContext()
}

type VariablePathContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariablePathContext() *VariablePathContext {
	var p = new(VariablePathContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_variablePath
	return p
}

func (*VariablePathContext) IsVariablePathContext() {}

func NewVariablePathContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariablePathContext {
	var p = new(VariablePathContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_variablePath

	return p
}

func (s *VariablePathContext) GetParser() antlr.Parser { return s.parser }

func (s *VariablePathContext) DOT() antlr.TerminalNode {
	return s.GetToken(ActionParserDOT, 0)
}

func (s *VariablePathContext) VariableIdentifier() IVariableIdentifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableIdentifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableIdentifierContext)
}

func (s *VariablePathContext) Array() IArrayContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArrayContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArrayContext)
}

func (s *VariablePathContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariablePathContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariablePathContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterVariablePath(s)
	}
}

func (s *VariablePathContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitVariablePath(s)
	}
}

func (p *ActionParser) VariablePath() (localctx IVariablePathContext) {
	this := p
	_ = this

	localctx = NewVariablePathContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, ActionParserRULE_variablePath)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	p.SetState(113)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case ActionParserDOT:
		{
			p.SetState(110)
			p.Match(ActionParserDOT)
		}
		{
			p.SetState(111)
			p.VariableIdentifier()
		}

	case ActionParserT__1:
		{
			p.SetState(112)
			p.Array()
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
	}

	return localctx
}

// IVariableIdentifierContext is an interface to support dynamic dispatch.
type IVariableIdentifierContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ID() antlr.TerminalNode

	// IsVariableIdentifierContext differentiates from other interfaces.
	IsVariableIdentifierContext()
}

type VariableIdentifierContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableIdentifierContext() *VariableIdentifierContext {
	var p = new(VariableIdentifierContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_variableIdentifier
	return p
}

func (*VariableIdentifierContext) IsVariableIdentifierContext() {}

func NewVariableIdentifierContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableIdentifierContext {
	var p = new(VariableIdentifierContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_variableIdentifier

	return p
}

func (s *VariableIdentifierContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableIdentifierContext) ID() antlr.TerminalNode {
	return s.GetToken(ActionParserID, 0)
}

func (s *VariableIdentifierContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableIdentifierContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableIdentifierContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterVariableIdentifier(s)
	}
}

func (s *VariableIdentifierContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitVariableIdentifier(s)
	}
}

func (p *ActionParser) VariableIdentifier() (localctx IVariableIdentifierContext) {
	this := p
	_ = this

	localctx = NewVariableIdentifierContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, ActionParserRULE_variableIdentifier)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(115)
		p.Match(ActionParserID)
	}

	return localctx
}

// INumberExpressionContext is an interface to support dynamic dispatch.
type INumberExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NUMBER() antlr.TerminalNode

	// IsNumberExpressionContext differentiates from other interfaces.
	IsNumberExpressionContext()
}

type NumberExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNumberExpressionContext() *NumberExpressionContext {
	var p = new(NumberExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_numberExpression
	return p
}

func (*NumberExpressionContext) IsNumberExpressionContext() {}

func NewNumberExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NumberExpressionContext {
	var p = new(NumberExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_numberExpression

	return p
}

func (s *NumberExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *NumberExpressionContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(ActionParserNUMBER, 0)
}

func (s *NumberExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NumberExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NumberExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterNumberExpression(s)
	}
}

func (s *NumberExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitNumberExpression(s)
	}
}

func (p *ActionParser) NumberExpression() (localctx INumberExpressionContext) {
	this := p
	_ = this

	localctx = NewNumberExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, ActionParserRULE_numberExpression)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(117)
		p.Match(ActionParserNUMBER)
	}

	return localctx
}

// IStringExpressionContext is an interface to support dynamic dispatch.
type IStringExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	STRING_INSIDE_EXPRESSION() antlr.TerminalNode

	// IsStringExpressionContext differentiates from other interfaces.
	IsStringExpressionContext()
}

type StringExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStringExpressionContext() *StringExpressionContext {
	var p = new(StringExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_stringExpression
	return p
}

func (*StringExpressionContext) IsStringExpressionContext() {}

func NewStringExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StringExpressionContext {
	var p = new(StringExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_stringExpression

	return p
}

func (s *StringExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *StringExpressionContext) STRING_INSIDE_EXPRESSION() antlr.TerminalNode {
	return s.GetToken(ActionParserSTRING_INSIDE_EXPRESSION, 0)
}

func (s *StringExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StringExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StringExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterStringExpression(s)
	}
}

func (s *StringExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitStringExpression(s)
	}
}

func (p *ActionParser) StringExpression() (localctx IStringExpressionContext) {
	this := p
	_ = this

	localctx = NewStringExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, ActionParserRULE_stringExpression)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(119)
		p.Match(ActionParserSTRING_INSIDE_EXPRESSION)
	}

	return localctx
}

// ITermExpressionContext is an interface to support dynamic dispatch.
type ITermExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LPAREN() antlr.TerminalNode
	OrExpression() IOrExpressionContext
	RPAREN() antlr.TerminalNode

	// IsTermExpressionContext differentiates from other interfaces.
	IsTermExpressionContext()
}

type TermExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTermExpressionContext() *TermExpressionContext {
	var p = new(TermExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_termExpression
	return p
}

func (*TermExpressionContext) IsTermExpressionContext() {}

func NewTermExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TermExpressionContext {
	var p = new(TermExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_termExpression

	return p
}

func (s *TermExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *TermExpressionContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ActionParserLPAREN, 0)
}

func (s *TermExpressionContext) OrExpression() IOrExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOrExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOrExpressionContext)
}

func (s *TermExpressionContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ActionParserRPAREN, 0)
}

func (s *TermExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TermExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TermExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterTermExpression(s)
	}
}

func (s *TermExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitTermExpression(s)
	}
}

func (p *ActionParser) TermExpression() (localctx ITermExpressionContext) {
	this := p
	_ = this

	localctx = NewTermExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, ActionParserRULE_termExpression)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(121)
		p.Match(ActionParserLPAREN)
	}
	{
		p.SetState(122)
		p.OrExpression()
	}
	{
		p.SetState(123)
		p.Match(ActionParserRPAREN)
	}

	return localctx
}

// INotExpressionContext is an interface to support dynamic dispatch.
type INotExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NOT() antlr.TerminalNode
	PrimaryExpression() IPrimaryExpressionContext

	// IsNotExpressionContext differentiates from other interfaces.
	IsNotExpressionContext()
}

type NotExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNotExpressionContext() *NotExpressionContext {
	var p = new(NotExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_notExpression
	return p
}

func (*NotExpressionContext) IsNotExpressionContext() {}

func NewNotExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NotExpressionContext {
	var p = new(NotExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_notExpression

	return p
}

func (s *NotExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *NotExpressionContext) NOT() antlr.TerminalNode {
	return s.GetToken(ActionParserNOT, 0)
}

func (s *NotExpressionContext) PrimaryExpression() IPrimaryExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimaryExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimaryExpressionContext)
}

func (s *NotExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NotExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NotExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterNotExpression(s)
	}
}

func (s *NotExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitNotExpression(s)
	}
}

func (p *ActionParser) NotExpression() (localctx INotExpressionContext) {
	this := p
	_ = this

	localctx = NewNotExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, ActionParserRULE_notExpression)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(125)
		p.Match(ActionParserNOT)
	}
	{
		p.SetState(126)
		p.PrimaryExpression()
	}

	return localctx
}

// IFunctionCallContext is an interface to support dynamic dispatch.
type IFunctionCallContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	FunctionName() IFunctionNameContext
	LPAREN() antlr.TerminalNode
	AllFunctionCallArguments() []IFunctionCallArgumentsContext
	FunctionCallArguments(i int) IFunctionCallArgumentsContext
	RPAREN() antlr.TerminalNode

	// IsFunctionCallContext differentiates from other interfaces.
	IsFunctionCallContext()
}

type FunctionCallContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionCallContext() *FunctionCallContext {
	var p = new(FunctionCallContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_functionCall
	return p
}

func (*FunctionCallContext) IsFunctionCallContext() {}

func NewFunctionCallContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionCallContext {
	var p = new(FunctionCallContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_functionCall

	return p
}

func (s *FunctionCallContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionCallContext) FunctionName() IFunctionNameContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionNameContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionNameContext)
}

func (s *FunctionCallContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ActionParserLPAREN, 0)
}

func (s *FunctionCallContext) AllFunctionCallArguments() []IFunctionCallArgumentsContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFunctionCallArgumentsContext); ok {
			len++
		}
	}

	tst := make([]IFunctionCallArgumentsContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFunctionCallArgumentsContext); ok {
			tst[i] = t.(IFunctionCallArgumentsContext)
			i++
		}
	}

	return tst
}

func (s *FunctionCallContext) FunctionCallArguments(i int) IFunctionCallArgumentsContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionCallArgumentsContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionCallArgumentsContext)
}

func (s *FunctionCallContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ActionParserRPAREN, 0)
}

func (s *FunctionCallContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionCallContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionCallContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterFunctionCall(s)
	}
}

func (s *FunctionCallContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitFunctionCall(s)
	}
}

func (p *ActionParser) FunctionCall() (localctx IFunctionCallContext) {
	this := p
	_ = this

	localctx = NewFunctionCallContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, ActionParserRULE_functionCall)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(128)
		p.FunctionName()
	}
	{
		p.SetState(129)
		p.Match(ActionParserLPAREN)
	}
	{
		p.SetState(130)
		p.FunctionCallArguments()
	}
	p.SetState(135)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserT__0 {
		{
			p.SetState(131)
			p.Match(ActionParserT__0)
		}
		{
			p.SetState(132)
			p.FunctionCallArguments()
		}

		p.SetState(137)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(138)
		p.Match(ActionParserRPAREN)
	}

	return localctx
}

// IFunctionNameContext is an interface to support dynamic dispatch.
type IFunctionNameContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ID() antlr.TerminalNode

	// IsFunctionNameContext differentiates from other interfaces.
	IsFunctionNameContext()
}

type FunctionNameContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionNameContext() *FunctionNameContext {
	var p = new(FunctionNameContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_functionName
	return p
}

func (*FunctionNameContext) IsFunctionNameContext() {}

func NewFunctionNameContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionNameContext {
	var p = new(FunctionNameContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_functionName

	return p
}

func (s *FunctionNameContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionNameContext) ID() antlr.TerminalNode {
	return s.GetToken(ActionParserID, 0)
}

func (s *FunctionNameContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionNameContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionNameContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterFunctionName(s)
	}
}

func (s *FunctionNameContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitFunctionName(s)
	}
}

func (p *ActionParser) FunctionName() (localctx IFunctionNameContext) {
	this := p
	_ = this

	localctx = NewFunctionNameContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, ActionParserRULE_functionName)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(140)
		p.Match(ActionParserID)
	}

	return localctx
}

// IFunctionCallArgumentsContext is an interface to support dynamic dispatch.
type IFunctionCallArgumentsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VariableContext() IVariableContextContext
	NumberExpression() INumberExpressionContext
	Literal() ILiteralContext

	// IsFunctionCallArgumentsContext differentiates from other interfaces.
	IsFunctionCallArgumentsContext()
}

type FunctionCallArgumentsContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionCallArgumentsContext() *FunctionCallArgumentsContext {
	var p = new(FunctionCallArgumentsContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_functionCallArguments
	return p
}

func (*FunctionCallArgumentsContext) IsFunctionCallArgumentsContext() {}

func NewFunctionCallArgumentsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionCallArgumentsContext {
	var p = new(FunctionCallArgumentsContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_functionCallArguments

	return p
}

func (s *FunctionCallArgumentsContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionCallArgumentsContext) VariableContext() IVariableContextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableContextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableContextContext)
}

func (s *FunctionCallArgumentsContext) NumberExpression() INumberExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INumberExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INumberExpressionContext)
}

func (s *FunctionCallArgumentsContext) Literal() ILiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILiteralContext)
}

func (s *FunctionCallArgumentsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionCallArgumentsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionCallArgumentsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterFunctionCallArguments(s)
	}
}

func (s *FunctionCallArgumentsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitFunctionCallArguments(s)
	}
}

func (p *ActionParser) FunctionCallArguments() (localctx IFunctionCallArgumentsContext) {
	this := p
	_ = this

	localctx = NewFunctionCallArgumentsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, ActionParserRULE_functionCallArguments)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.SetState(146)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 9, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(143)
			p.VariableContext()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(144)
			p.NumberExpression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(145)
			p.Literal()
		}

	}

	return localctx
}

// IArrayContext is an interface to support dynamic dispatch.
type IArrayContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ArrayIndex() IArrayIndexContext

	// IsArrayContext differentiates from other interfaces.
	IsArrayContext()
}

type ArrayContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArrayContext() *ArrayContext {
	var p = new(ArrayContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_array
	return p
}

func (*ArrayContext) IsArrayContext() {}

func NewArrayContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArrayContext {
	var p = new(ArrayContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_array

	return p
}

func (s *ArrayContext) GetParser() antlr.Parser { return s.parser }

func (s *ArrayContext) ArrayIndex() IArrayIndexContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArrayIndexContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArrayIndexContext)
}

func (s *ArrayContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArrayContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArrayContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterArray(s)
	}
}

func (s *ArrayContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitArray(s)
	}
}

func (p *ActionParser) Array() (localctx IArrayContext) {
	this := p
	_ = this

	localctx = NewArrayContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, ActionParserRULE_array)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(148)
		p.Match(ActionParserT__1)
	}
	{
		p.SetState(149)
		p.ArrayIndex()
	}
	{
		p.SetState(150)
		p.Match(ActionParserT__2)
	}

	return localctx
}

// IArrayIndexContext is an interface to support dynamic dispatch.
type IArrayIndexContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PrimaryExpression() IPrimaryExpressionContext

	// IsArrayIndexContext differentiates from other interfaces.
	IsArrayIndexContext()
}

type ArrayIndexContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArrayIndexContext() *ArrayIndexContext {
	var p = new(ArrayIndexContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_arrayIndex
	return p
}

func (*ArrayIndexContext) IsArrayIndexContext() {}

func NewArrayIndexContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArrayIndexContext {
	var p = new(ArrayIndexContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_arrayIndex

	return p
}

func (s *ArrayIndexContext) GetParser() antlr.Parser { return s.parser }

func (s *ArrayIndexContext) PrimaryExpression() IPrimaryExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimaryExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimaryExpressionContext)
}

func (s *ArrayIndexContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArrayIndexContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArrayIndexContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterArrayIndex(s)
	}
}

func (s *ArrayIndexContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitArrayIndex(s)
	}
}

func (p *ActionParser) ArrayIndex() (localctx IArrayIndexContext) {
	this := p
	_ = this

	localctx = NewArrayIndexContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, ActionParserRULE_arrayIndex)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(152)
		p.PrimaryExpression()
	}

	return localctx
}

// IAndOperatorContext is an interface to support dynamic dispatch.
type IAndOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AND() antlr.TerminalNode

	// IsAndOperatorContext differentiates from other interfaces.
	IsAndOperatorContext()
}

type AndOperatorContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAndOperatorContext() *AndOperatorContext {
	var p = new(AndOperatorContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_andOperator
	return p
}

func (*AndOperatorContext) IsAndOperatorContext() {}

func NewAndOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AndOperatorContext {
	var p = new(AndOperatorContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_andOperator

	return p
}

func (s *AndOperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *AndOperatorContext) AND() antlr.TerminalNode {
	return s.GetToken(ActionParserAND, 0)
}

func (s *AndOperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AndOperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AndOperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterAndOperator(s)
	}
}

func (s *AndOperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitAndOperator(s)
	}
}

func (p *ActionParser) AndOperator() (localctx IAndOperatorContext) {
	this := p
	_ = this

	localctx = NewAndOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, ActionParserRULE_andOperator)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(154)
		p.Match(ActionParserAND)
	}

	return localctx
}

// IOrOperatorContext is an interface to support dynamic dispatch.
type IOrOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	OR() antlr.TerminalNode

	// IsOrOperatorContext differentiates from other interfaces.
	IsOrOperatorContext()
}

type OrOperatorContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOrOperatorContext() *OrOperatorContext {
	var p = new(OrOperatorContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_orOperator
	return p
}

func (*OrOperatorContext) IsOrOperatorContext() {}

func NewOrOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OrOperatorContext {
	var p = new(OrOperatorContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_orOperator

	return p
}

func (s *OrOperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *OrOperatorContext) OR() antlr.TerminalNode {
	return s.GetToken(ActionParserOR, 0)
}

func (s *OrOperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OrOperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OrOperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterOrOperator(s)
	}
}

func (s *OrOperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitOrOperator(s)
	}
}

func (p *ActionParser) OrOperator() (localctx IOrOperatorContext) {
	this := p
	_ = this

	localctx = NewOrOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 40, ActionParserRULE_orOperator)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(156)
		p.Match(ActionParserOR)
	}

	return localctx
}

// IComparisonOperatorContext is an interface to support dynamic dispatch.
type IComparisonOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	GT() antlr.TerminalNode
	LT() antlr.TerminalNode
	GTE() antlr.TerminalNode
	LTE() antlr.TerminalNode

	// IsComparisonOperatorContext differentiates from other interfaces.
	IsComparisonOperatorContext()
}

type ComparisonOperatorContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyComparisonOperatorContext() *ComparisonOperatorContext {
	var p = new(ComparisonOperatorContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_comparisonOperator
	return p
}

func (*ComparisonOperatorContext) IsComparisonOperatorContext() {}

func NewComparisonOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ComparisonOperatorContext {
	var p = new(ComparisonOperatorContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_comparisonOperator

	return p
}

func (s *ComparisonOperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *ComparisonOperatorContext) GT() antlr.TerminalNode {
	return s.GetToken(ActionParserGT, 0)
}

func (s *ComparisonOperatorContext) LT() antlr.TerminalNode {
	return s.GetToken(ActionParserLT, 0)
}

func (s *ComparisonOperatorContext) GTE() antlr.TerminalNode {
	return s.GetToken(ActionParserGTE, 0)
}

func (s *ComparisonOperatorContext) LTE() antlr.TerminalNode {
	return s.GetToken(ActionParserLTE, 0)
}

func (s *ComparisonOperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ComparisonOperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ComparisonOperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterComparisonOperator(s)
	}
}

func (s *ComparisonOperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitComparisonOperator(s)
	}
}

func (p *ActionParser) ComparisonOperator() (localctx IComparisonOperatorContext) {
	this := p
	_ = this

	localctx = NewComparisonOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 42, ActionParserRULE_comparisonOperator)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(158)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&61440) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

	return localctx
}

// IEqualityOperatorContext is an interface to support dynamic dispatch.
type IEqualityOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EQ() antlr.TerminalNode
	NEQ() antlr.TerminalNode

	// IsEqualityOperatorContext differentiates from other interfaces.
	IsEqualityOperatorContext()
}

type EqualityOperatorContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEqualityOperatorContext() *EqualityOperatorContext {
	var p = new(EqualityOperatorContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_equalityOperator
	return p
}

func (*EqualityOperatorContext) IsEqualityOperatorContext() {}

func NewEqualityOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EqualityOperatorContext {
	var p = new(EqualityOperatorContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_equalityOperator

	return p
}

func (s *EqualityOperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *EqualityOperatorContext) EQ() antlr.TerminalNode {
	return s.GetToken(ActionParserEQ, 0)
}

func (s *EqualityOperatorContext) NEQ() antlr.TerminalNode {
	return s.GetToken(ActionParserNEQ, 0)
}

func (s *EqualityOperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EqualityOperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EqualityOperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterEqualityOperator(s)
	}
}

func (s *EqualityOperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitEqualityOperator(s)
	}
}

func (p *ActionParser) EqualityOperator() (localctx IEqualityOperatorContext) {
	this := p
	_ = this

	localctx = NewEqualityOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 44, ActionParserRULE_equalityOperator)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(160)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ActionParserEQ || _la == ActionParserNEQ) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

	return localctx
}

// ILiteralContext is an interface to support dynamic dispatch.
type ILiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	STRING_INSIDE_EXPRESSION() antlr.TerminalNode
	BOOLEAN() antlr.TerminalNode
	NULL() antlr.TerminalNode
	NUMBER() antlr.TerminalNode

	// IsLiteralContext differentiates from other interfaces.
	IsLiteralContext()
}

type LiteralContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLiteralContext() *LiteralContext {
	var p = new(LiteralContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_literal
	return p
}

func (*LiteralContext) IsLiteralContext() {}

func NewLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LiteralContext {
	var p = new(LiteralContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_literal

	return p
}

func (s *LiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *LiteralContext) STRING_INSIDE_EXPRESSION() antlr.TerminalNode {
	return s.GetToken(ActionParserSTRING_INSIDE_EXPRESSION, 0)
}

func (s *LiteralContext) BOOLEAN() antlr.TerminalNode {
	return s.GetToken(ActionParserBOOLEAN, 0)
}

func (s *LiteralContext) NULL() antlr.TerminalNode {
	return s.GetToken(ActionParserNULL, 0)
}

func (s *LiteralContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(ActionParserNUMBER, 0)
}

func (s *LiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LiteralContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterLiteral(s)
	}
}

func (s *LiteralContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitLiteral(s)
	}
}

func (p *ActionParser) Literal() (localctx ILiteralContext) {
	this := p
	_ = this

	localctx = NewLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 46, ActionParserRULE_literal)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(162)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&624) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

	return localctx
}

// IExpressionStartContext is an interface to support dynamic dispatch.
type IExpressionStartContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EXP_START() antlr.TerminalNode

	// IsExpressionStartContext differentiates from other interfaces.
	IsExpressionStartContext()
}

type ExpressionStartContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionStartContext() *ExpressionStartContext {
	var p = new(ExpressionStartContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_expressionStart
	return p
}

func (*ExpressionStartContext) IsExpressionStartContext() {}

func NewExpressionStartContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionStartContext {
	var p = new(ExpressionStartContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_expressionStart

	return p
}

func (s *ExpressionStartContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionStartContext) EXP_START() antlr.TerminalNode {
	return s.GetToken(ActionParserEXP_START, 0)
}

func (s *ExpressionStartContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionStartContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionStartContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterExpressionStart(s)
	}
}

func (s *ExpressionStartContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitExpressionStart(s)
	}
}

func (p *ActionParser) ExpressionStart() (localctx IExpressionStartContext) {
	this := p
	_ = this

	localctx = NewExpressionStartContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, ActionParserRULE_expressionStart)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(164)
		p.Match(ActionParserEXP_START)
	}

	return localctx
}

// IExpressionEndContext is an interface to support dynamic dispatch.
type IExpressionEndContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EXP_END() antlr.TerminalNode

	// IsExpressionEndContext differentiates from other interfaces.
	IsExpressionEndContext()
}

type ExpressionEndContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionEndContext() *ExpressionEndContext {
	var p = new(ExpressionEndContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_expressionEnd
	return p
}

func (*ExpressionEndContext) IsExpressionEndContext() {}

func NewExpressionEndContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionEndContext {
	var p = new(ExpressionEndContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_expressionEnd

	return p
}

func (s *ExpressionEndContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionEndContext) EXP_END() antlr.TerminalNode {
	return s.GetToken(ActionParserEXP_END, 0)
}

func (s *ExpressionEndContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionEndContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionEndContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterExpressionEnd(s)
	}
}

func (s *ExpressionEndContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitExpressionEnd(s)
	}
}

func (p *ActionParser) ExpressionEnd() (localctx IExpressionEndContext) {
	this := p
	_ = this

	localctx = NewExpressionEndContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 50, ActionParserRULE_expressionEnd)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(166)
		p.Match(ActionParserEXP_END)
	}

	return localctx
}
