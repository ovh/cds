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
		"'&&'", "'.'", "'*'",
	}
	staticData.symbolicNames = []string{
		"", "", "", "", "STRING_INSIDE_EXPRESSION", "BOOLEAN", "NULL", "EXP_START",
		"EXP_END", "NUMBER", "EQ", "NEQ", "GT", "LT", "GTE", "LTE", "ID", "LPAREN",
		"RPAREN", "NOT", "OR", "AND", "DOT", "STAR", "WS",
	}
	staticData.ruleNames = []string{
		"start", "expression", "orExpression", "andExpression", "comparisonExpression",
		"equalityExpression", "primaryExpression", "variableContext", "variablePath",
		"variableIdentifier", "numberExpression", "stringExpression", "termExpression",
		"notExpression", "notOperator", "functionCall", "functionName", "functionCallArguments",
		"array", "arrayIndex", "andOperator", "orOperator", "comparisonOperator",
		"equalityOperator", "booleanExpression", "expressionStart", "expressionEnd",
		"filterExpression",
	}
	staticData.predictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 24, 180, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 5, 1, 63, 8, 1,
		10, 1, 12, 1, 66, 9, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 2, 5, 2, 74, 8,
		2, 10, 2, 12, 2, 77, 9, 2, 1, 3, 1, 3, 1, 3, 1, 3, 5, 3, 83, 8, 3, 10,
		3, 12, 3, 86, 9, 3, 1, 4, 1, 4, 1, 4, 1, 4, 3, 4, 92, 8, 4, 1, 5, 1, 5,
		1, 5, 1, 5, 3, 5, 98, 8, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 3, 6, 106,
		8, 6, 1, 7, 1, 7, 5, 7, 110, 8, 7, 10, 7, 12, 7, 113, 9, 7, 1, 8, 1, 8,
		1, 8, 1, 8, 1, 8, 3, 8, 120, 8, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1,
		11, 1, 12, 1, 12, 1, 12, 1, 12, 1, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 15,
		1, 15, 1, 15, 1, 15, 1, 15, 5, 15, 142, 8, 15, 10, 15, 12, 15, 145, 9,
		15, 1, 15, 1, 15, 1, 16, 1, 16, 1, 17, 1, 17, 1, 17, 1, 17, 1, 17, 3, 17,
		156, 8, 17, 1, 18, 1, 18, 1, 18, 1, 18, 1, 19, 1, 19, 1, 20, 1, 20, 1,
		21, 1, 21, 1, 22, 1, 22, 1, 23, 1, 23, 1, 24, 1, 24, 1, 25, 1, 25, 1, 26,
		1, 26, 1, 27, 1, 27, 1, 27, 0, 0, 28, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18,
		20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54,
		0, 2, 1, 0, 12, 15, 1, 0, 10, 11, 169, 0, 56, 1, 0, 0, 0, 2, 59, 1, 0,
		0, 0, 4, 69, 1, 0, 0, 0, 6, 78, 1, 0, 0, 0, 8, 87, 1, 0, 0, 0, 10, 93,
		1, 0, 0, 0, 12, 105, 1, 0, 0, 0, 14, 107, 1, 0, 0, 0, 16, 119, 1, 0, 0,
		0, 18, 121, 1, 0, 0, 0, 20, 123, 1, 0, 0, 0, 22, 125, 1, 0, 0, 0, 24, 127,
		1, 0, 0, 0, 26, 131, 1, 0, 0, 0, 28, 134, 1, 0, 0, 0, 30, 136, 1, 0, 0,
		0, 32, 148, 1, 0, 0, 0, 34, 155, 1, 0, 0, 0, 36, 157, 1, 0, 0, 0, 38, 161,
		1, 0, 0, 0, 40, 163, 1, 0, 0, 0, 42, 165, 1, 0, 0, 0, 44, 167, 1, 0, 0,
		0, 46, 169, 1, 0, 0, 0, 48, 171, 1, 0, 0, 0, 50, 173, 1, 0, 0, 0, 52, 175,
		1, 0, 0, 0, 54, 177, 1, 0, 0, 0, 56, 57, 3, 2, 1, 0, 57, 58, 5, 0, 0, 1,
		58, 1, 1, 0, 0, 0, 59, 60, 3, 50, 25, 0, 60, 64, 3, 4, 2, 0, 61, 63, 3,
		4, 2, 0, 62, 61, 1, 0, 0, 0, 63, 66, 1, 0, 0, 0, 64, 62, 1, 0, 0, 0, 64,
		65, 1, 0, 0, 0, 65, 67, 1, 0, 0, 0, 66, 64, 1, 0, 0, 0, 67, 68, 3, 52,
		26, 0, 68, 3, 1, 0, 0, 0, 69, 75, 3, 6, 3, 0, 70, 71, 3, 42, 21, 0, 71,
		72, 3, 6, 3, 0, 72, 74, 1, 0, 0, 0, 73, 70, 1, 0, 0, 0, 74, 77, 1, 0, 0,
		0, 75, 73, 1, 0, 0, 0, 75, 76, 1, 0, 0, 0, 76, 5, 1, 0, 0, 0, 77, 75, 1,
		0, 0, 0, 78, 84, 3, 8, 4, 0, 79, 80, 3, 40, 20, 0, 80, 81, 3, 8, 4, 0,
		81, 83, 1, 0, 0, 0, 82, 79, 1, 0, 0, 0, 83, 86, 1, 0, 0, 0, 84, 82, 1,
		0, 0, 0, 84, 85, 1, 0, 0, 0, 85, 7, 1, 0, 0, 0, 86, 84, 1, 0, 0, 0, 87,
		91, 3, 10, 5, 0, 88, 89, 3, 44, 22, 0, 89, 90, 3, 10, 5, 0, 90, 92, 1,
		0, 0, 0, 91, 88, 1, 0, 0, 0, 91, 92, 1, 0, 0, 0, 92, 9, 1, 0, 0, 0, 93,
		97, 3, 12, 6, 0, 94, 95, 3, 46, 23, 0, 95, 96, 3, 12, 6, 0, 96, 98, 1,
		0, 0, 0, 97, 94, 1, 0, 0, 0, 97, 98, 1, 0, 0, 0, 98, 11, 1, 0, 0, 0, 99,
		106, 3, 14, 7, 0, 100, 106, 3, 20, 10, 0, 101, 106, 3, 30, 15, 0, 102,
		106, 3, 22, 11, 0, 103, 106, 3, 24, 12, 0, 104, 106, 3, 26, 13, 0, 105,
		99, 1, 0, 0, 0, 105, 100, 1, 0, 0, 0, 105, 101, 1, 0, 0, 0, 105, 102, 1,
		0, 0, 0, 105, 103, 1, 0, 0, 0, 105, 104, 1, 0, 0, 0, 106, 13, 1, 0, 0,
		0, 107, 111, 3, 18, 9, 0, 108, 110, 3, 16, 8, 0, 109, 108, 1, 0, 0, 0,
		110, 113, 1, 0, 0, 0, 111, 109, 1, 0, 0, 0, 111, 112, 1, 0, 0, 0, 112,
		15, 1, 0, 0, 0, 113, 111, 1, 0, 0, 0, 114, 115, 5, 22, 0, 0, 115, 120,
		3, 18, 9, 0, 116, 120, 3, 36, 18, 0, 117, 118, 5, 22, 0, 0, 118, 120, 3,
		54, 27, 0, 119, 114, 1, 0, 0, 0, 119, 116, 1, 0, 0, 0, 119, 117, 1, 0,
		0, 0, 120, 17, 1, 0, 0, 0, 121, 122, 5, 16, 0, 0, 122, 19, 1, 0, 0, 0,
		123, 124, 5, 9, 0, 0, 124, 21, 1, 0, 0, 0, 125, 126, 5, 4, 0, 0, 126, 23,
		1, 0, 0, 0, 127, 128, 5, 17, 0, 0, 128, 129, 3, 4, 2, 0, 129, 130, 5, 18,
		0, 0, 130, 25, 1, 0, 0, 0, 131, 132, 3, 28, 14, 0, 132, 133, 3, 12, 6,
		0, 133, 27, 1, 0, 0, 0, 134, 135, 5, 19, 0, 0, 135, 29, 1, 0, 0, 0, 136,
		137, 3, 32, 16, 0, 137, 138, 5, 17, 0, 0, 138, 143, 3, 34, 17, 0, 139,
		140, 5, 1, 0, 0, 140, 142, 3, 34, 17, 0, 141, 139, 1, 0, 0, 0, 142, 145,
		1, 0, 0, 0, 143, 141, 1, 0, 0, 0, 143, 144, 1, 0, 0, 0, 144, 146, 1, 0,
		0, 0, 145, 143, 1, 0, 0, 0, 146, 147, 5, 18, 0, 0, 147, 31, 1, 0, 0, 0,
		148, 149, 5, 16, 0, 0, 149, 33, 1, 0, 0, 0, 150, 156, 1, 0, 0, 0, 151,
		156, 3, 14, 7, 0, 152, 156, 3, 22, 11, 0, 153, 156, 3, 20, 10, 0, 154,
		156, 3, 48, 24, 0, 155, 150, 1, 0, 0, 0, 155, 151, 1, 0, 0, 0, 155, 152,
		1, 0, 0, 0, 155, 153, 1, 0, 0, 0, 155, 154, 1, 0, 0, 0, 156, 35, 1, 0,
		0, 0, 157, 158, 5, 2, 0, 0, 158, 159, 3, 38, 19, 0, 159, 160, 5, 3, 0,
		0, 160, 37, 1, 0, 0, 0, 161, 162, 3, 12, 6, 0, 162, 39, 1, 0, 0, 0, 163,
		164, 5, 21, 0, 0, 164, 41, 1, 0, 0, 0, 165, 166, 5, 20, 0, 0, 166, 43,
		1, 0, 0, 0, 167, 168, 7, 0, 0, 0, 168, 45, 1, 0, 0, 0, 169, 170, 7, 1,
		0, 0, 170, 47, 1, 0, 0, 0, 171, 172, 5, 5, 0, 0, 172, 49, 1, 0, 0, 0, 173,
		174, 5, 7, 0, 0, 174, 51, 1, 0, 0, 0, 175, 176, 5, 8, 0, 0, 176, 53, 1,
		0, 0, 0, 177, 178, 5, 23, 0, 0, 178, 55, 1, 0, 0, 0, 10, 64, 75, 84, 91,
		97, 105, 111, 119, 143, 155,
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
	ActionParserSTAR                     = 23
	ActionParserWS                       = 24
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
	ActionParserRULE_notOperator           = 14
	ActionParserRULE_functionCall          = 15
	ActionParserRULE_functionName          = 16
	ActionParserRULE_functionCallArguments = 17
	ActionParserRULE_array                 = 18
	ActionParserRULE_arrayIndex            = 19
	ActionParserRULE_andOperator           = 20
	ActionParserRULE_orOperator            = 21
	ActionParserRULE_comparisonOperator    = 22
	ActionParserRULE_equalityOperator      = 23
	ActionParserRULE_booleanExpression     = 24
	ActionParserRULE_expressionStart       = 25
	ActionParserRULE_expressionEnd         = 26
	ActionParserRULE_filterExpression      = 27
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
		p.SetState(56)
		p.Expression()
	}
	{
		p.SetState(57)
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
		p.SetState(59)
		p.ExpressionStart()
	}
	{
		p.SetState(60)
		p.OrExpression()
	}
	p.SetState(64)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&721424) != 0 {
		{
			p.SetState(61)
			p.OrExpression()
		}

		p.SetState(66)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(67)
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
		p.SetState(69)
		p.AndExpression()
	}
	p.SetState(75)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserOR {
		{
			p.SetState(70)
			p.OrOperator()
		}
		{
			p.SetState(71)
			p.AndExpression()
		}

		p.SetState(77)
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
		p.SetState(78)
		p.ComparisonExpression()
	}
	p.SetState(84)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserAND {
		{
			p.SetState(79)
			p.AndOperator()
		}
		{
			p.SetState(80)
			p.ComparisonExpression()
		}

		p.SetState(86)
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
		p.SetState(87)
		p.EqualityExpression()
	}
	p.SetState(91)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&61440) != 0 {
		{
			p.SetState(88)
			p.ComparisonOperator()
		}
		{
			p.SetState(89)
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
		p.SetState(93)
		p.PrimaryExpression()
	}
	p.SetState(97)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == ActionParserEQ || _la == ActionParserNEQ {
		{
			p.SetState(94)
			p.EqualityOperator()
		}
		{
			p.SetState(95)
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

	p.SetState(105)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 5, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(99)
			p.VariableContext()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(100)
			p.NumberExpression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(101)
			p.FunctionCall()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(102)
			p.StringExpression()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(103)
			p.TermExpression()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(104)
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
		p.SetState(107)
		p.VariableIdentifier()
	}
	p.SetState(111)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserT__1 || _la == ActionParserDOT {
		{
			p.SetState(108)
			p.VariablePath()
		}

		p.SetState(113)
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
	FilterExpression() IFilterExpressionContext

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

func (s *VariablePathContext) FilterExpression() IFilterExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFilterExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFilterExpressionContext)
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
	p.SetState(119)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 7, p.GetParserRuleContext()) {
	case 1:
		{
			p.SetState(114)
			p.Match(ActionParserDOT)
		}
		{
			p.SetState(115)
			p.VariableIdentifier()
		}

	case 2:
		{
			p.SetState(116)
			p.Array()
		}

	case 3:
		{
			p.SetState(117)
			p.Match(ActionParserDOT)
		}
		{
			p.SetState(118)
			p.FilterExpression()
		}

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
		p.SetState(121)
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
		p.SetState(123)
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
		p.SetState(125)
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
		p.SetState(127)
		p.Match(ActionParserLPAREN)
	}
	{
		p.SetState(128)
		p.OrExpression()
	}
	{
		p.SetState(129)
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
	NotOperator() INotOperatorContext
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

func (s *NotExpressionContext) NotOperator() INotOperatorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INotOperatorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INotOperatorContext)
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
		p.SetState(131)
		p.NotOperator()
	}
	{
		p.SetState(132)
		p.PrimaryExpression()
	}

	return localctx
}

// INotOperatorContext is an interface to support dynamic dispatch.
type INotOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NOT() antlr.TerminalNode

	// IsNotOperatorContext differentiates from other interfaces.
	IsNotOperatorContext()
}

type NotOperatorContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNotOperatorContext() *NotOperatorContext {
	var p = new(NotOperatorContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_notOperator
	return p
}

func (*NotOperatorContext) IsNotOperatorContext() {}

func NewNotOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NotOperatorContext {
	var p = new(NotOperatorContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_notOperator

	return p
}

func (s *NotOperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *NotOperatorContext) NOT() antlr.TerminalNode {
	return s.GetToken(ActionParserNOT, 0)
}

func (s *NotOperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NotOperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NotOperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterNotOperator(s)
	}
}

func (s *NotOperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitNotOperator(s)
	}
}

func (p *ActionParser) NotOperator() (localctx INotOperatorContext) {
	this := p
	_ = this

	localctx = NewNotOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, ActionParserRULE_notOperator)

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
		p.SetState(134)
		p.Match(ActionParserNOT)
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
	p.EnterRule(localctx, 30, ActionParserRULE_functionCall)
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
		p.SetState(136)
		p.FunctionName()
	}
	{
		p.SetState(137)
		p.Match(ActionParserLPAREN)
	}
	{
		p.SetState(138)
		p.FunctionCallArguments()
	}
	p.SetState(143)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == ActionParserT__0 {
		{
			p.SetState(139)
			p.Match(ActionParserT__0)
		}
		{
			p.SetState(140)
			p.FunctionCallArguments()
		}

		p.SetState(145)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(146)
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
	p.EnterRule(localctx, 32, ActionParserRULE_functionName)

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
	StringExpression() IStringExpressionContext
	NumberExpression() INumberExpressionContext
	BooleanExpression() IBooleanExpressionContext

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

func (s *FunctionCallArgumentsContext) StringExpression() IStringExpressionContext {
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

func (s *FunctionCallArgumentsContext) BooleanExpression() IBooleanExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBooleanExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBooleanExpressionContext)
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
	p.EnterRule(localctx, 34, ActionParserRULE_functionCallArguments)

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

	p.SetState(155)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case ActionParserT__0, ActionParserRPAREN:
		p.EnterOuterAlt(localctx, 1)

	case ActionParserID:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(151)
			p.VariableContext()
		}

	case ActionParserSTRING_INSIDE_EXPRESSION:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(152)
			p.StringExpression()
		}

	case ActionParserNUMBER:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(153)
			p.NumberExpression()
		}

	case ActionParserBOOLEAN:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(154)
			p.BooleanExpression()
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
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
	p.EnterRule(localctx, 36, ActionParserRULE_array)

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
		p.SetState(157)
		p.Match(ActionParserT__1)
	}
	{
		p.SetState(158)
		p.ArrayIndex()
	}
	{
		p.SetState(159)
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
	p.EnterRule(localctx, 38, ActionParserRULE_arrayIndex)

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
		p.SetState(161)
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
	p.EnterRule(localctx, 40, ActionParserRULE_andOperator)

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
		p.SetState(163)
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
	p.EnterRule(localctx, 42, ActionParserRULE_orOperator)

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
		p.SetState(165)
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
	p.EnterRule(localctx, 44, ActionParserRULE_comparisonOperator)
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
		p.SetState(167)
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
	p.EnterRule(localctx, 46, ActionParserRULE_equalityOperator)
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
		p.SetState(169)
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

// IBooleanExpressionContext is an interface to support dynamic dispatch.
type IBooleanExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	BOOLEAN() antlr.TerminalNode

	// IsBooleanExpressionContext differentiates from other interfaces.
	IsBooleanExpressionContext()
}

type BooleanExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBooleanExpressionContext() *BooleanExpressionContext {
	var p = new(BooleanExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_booleanExpression
	return p
}

func (*BooleanExpressionContext) IsBooleanExpressionContext() {}

func NewBooleanExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BooleanExpressionContext {
	var p = new(BooleanExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_booleanExpression

	return p
}

func (s *BooleanExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *BooleanExpressionContext) BOOLEAN() antlr.TerminalNode {
	return s.GetToken(ActionParserBOOLEAN, 0)
}

func (s *BooleanExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BooleanExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BooleanExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterBooleanExpression(s)
	}
}

func (s *BooleanExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitBooleanExpression(s)
	}
}

func (p *ActionParser) BooleanExpression() (localctx IBooleanExpressionContext) {
	this := p
	_ = this

	localctx = NewBooleanExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, ActionParserRULE_booleanExpression)

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
		p.SetState(171)
		p.Match(ActionParserBOOLEAN)
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
	p.EnterRule(localctx, 50, ActionParserRULE_expressionStart)

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
		p.SetState(173)
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
	p.EnterRule(localctx, 52, ActionParserRULE_expressionEnd)

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
		p.SetState(175)
		p.Match(ActionParserEXP_END)
	}

	return localctx
}

// IFilterExpressionContext is an interface to support dynamic dispatch.
type IFilterExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	STAR() antlr.TerminalNode

	// IsFilterExpressionContext differentiates from other interfaces.
	IsFilterExpressionContext()
}

type FilterExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFilterExpressionContext() *FilterExpressionContext {
	var p = new(FilterExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = ActionParserRULE_filterExpression
	return p
}

func (*FilterExpressionContext) IsFilterExpressionContext() {}

func NewFilterExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FilterExpressionContext {
	var p = new(FilterExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = ActionParserRULE_filterExpression

	return p
}

func (s *FilterExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *FilterExpressionContext) STAR() antlr.TerminalNode {
	return s.GetToken(ActionParserSTAR, 0)
}

func (s *FilterExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FilterExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FilterExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.EnterFilterExpression(s)
	}
}

func (s *FilterExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ActionListener); ok {
		listenerT.ExitFilterExpression(s)
	}
}

func (p *ActionParser) FilterExpression() (localctx IFilterExpressionContext) {
	this := p
	_ = this

	localctx = NewFilterExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 54, ActionParserRULE_filterExpression)

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
		p.SetState(177)
		p.Match(ActionParserSTAR)
	}

	return localctx
}
