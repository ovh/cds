// Code generated from Action.g4 by ANTLR 4.12.0. DO NOT EDIT.

package parser

import (
	"fmt"
	"sync"
	"unicode"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type ActionLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var actionlexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	channelNames           []string
	modeNames              []string
	literalNames           []string
	symbolicNames          []string
	ruleNames              []string
	predictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func actionlexerLexerInit() {
	staticData := &actionlexerLexerStaticData
	staticData.channelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.modeNames = []string{
		"DEFAULT_MODE",
	}
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
		"T__0", "T__1", "T__2", "STRING_INSIDE_EXPRESSION", "BOOLEAN", "NULL",
		"EXP_START", "EXP_END", "NUMBER", "EQ", "NEQ", "GT", "LT", "GTE", "LTE",
		"ID", "LPAREN", "RPAREN", "NOT", "OR", "AND", "DOT", "ESC", "INT", "FLOAT",
		"EXPONENT", "IDENTIFIER", "WS",
	}
	staticData.predictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 23, 210, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1,
		3, 1, 3, 5, 3, 67, 8, 3, 10, 3, 12, 3, 70, 9, 3, 1, 3, 1, 3, 1, 4, 1, 4,
		1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 3, 4, 83, 8, 4, 1, 5, 1, 5, 1,
		5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 8, 1, 8, 3,
		8, 99, 8, 8, 1, 9, 1, 9, 1, 9, 1, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 12,
		1, 12, 1, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1,
		16, 1, 17, 1, 17, 1, 18, 1, 18, 1, 19, 1, 19, 1, 19, 1, 20, 1, 20, 1, 20,
		1, 21, 1, 21, 1, 22, 1, 22, 1, 22, 1, 23, 1, 23, 1, 23, 5, 23, 139, 8,
		23, 10, 23, 12, 23, 142, 9, 23, 3, 23, 144, 8, 23, 1, 24, 1, 24, 1, 24,
		5, 24, 149, 8, 24, 10, 24, 12, 24, 152, 9, 24, 3, 24, 154, 8, 24, 1, 24,
		1, 24, 5, 24, 158, 8, 24, 10, 24, 12, 24, 161, 9, 24, 1, 24, 3, 24, 164,
		8, 24, 1, 24, 1, 24, 4, 24, 168, 8, 24, 11, 24, 12, 24, 169, 1, 24, 3,
		24, 173, 8, 24, 1, 24, 1, 24, 1, 24, 5, 24, 178, 8, 24, 10, 24, 12, 24,
		181, 9, 24, 3, 24, 183, 8, 24, 1, 24, 3, 24, 186, 8, 24, 1, 25, 1, 25,
		3, 25, 190, 8, 25, 1, 25, 4, 25, 193, 8, 25, 11, 25, 12, 25, 194, 1, 26,
		1, 26, 5, 26, 199, 8, 26, 10, 26, 12, 26, 202, 9, 26, 1, 27, 4, 27, 205,
		8, 27, 11, 27, 12, 27, 206, 1, 27, 1, 27, 1, 68, 0, 28, 1, 1, 3, 2, 5,
		3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25,
		13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43,
		22, 45, 0, 47, 0, 49, 0, 51, 0, 53, 0, 55, 23, 1, 0, 8, 9, 0, 34, 34, 39,
		39, 47, 47, 92, 92, 98, 98, 102, 102, 110, 110, 114, 114, 116, 116, 1,
		0, 49, 57, 1, 0, 48, 57, 2, 0, 69, 69, 101, 101, 2, 0, 43, 43, 45, 45,
		3, 0, 65, 90, 95, 95, 97, 122, 5, 0, 45, 45, 48, 57, 65, 90, 95, 95, 97,
		122, 3, 0, 9, 10, 13, 13, 32, 32, 224, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0,
		0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0,
		0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0,
		0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1,
		0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35,
		1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0, 0, 0,
		43, 1, 0, 0, 0, 0, 55, 1, 0, 0, 0, 1, 57, 1, 0, 0, 0, 3, 59, 1, 0, 0, 0,
		5, 61, 1, 0, 0, 0, 7, 63, 1, 0, 0, 0, 9, 82, 1, 0, 0, 0, 11, 84, 1, 0,
		0, 0, 13, 89, 1, 0, 0, 0, 15, 93, 1, 0, 0, 0, 17, 98, 1, 0, 0, 0, 19, 100,
		1, 0, 0, 0, 21, 103, 1, 0, 0, 0, 23, 106, 1, 0, 0, 0, 25, 108, 1, 0, 0,
		0, 27, 110, 1, 0, 0, 0, 29, 113, 1, 0, 0, 0, 31, 116, 1, 0, 0, 0, 33, 118,
		1, 0, 0, 0, 35, 120, 1, 0, 0, 0, 37, 122, 1, 0, 0, 0, 39, 124, 1, 0, 0,
		0, 41, 127, 1, 0, 0, 0, 43, 130, 1, 0, 0, 0, 45, 132, 1, 0, 0, 0, 47, 143,
		1, 0, 0, 0, 49, 185, 1, 0, 0, 0, 51, 187, 1, 0, 0, 0, 53, 196, 1, 0, 0,
		0, 55, 204, 1, 0, 0, 0, 57, 58, 5, 44, 0, 0, 58, 2, 1, 0, 0, 0, 59, 60,
		5, 91, 0, 0, 60, 4, 1, 0, 0, 0, 61, 62, 5, 93, 0, 0, 62, 6, 1, 0, 0, 0,
		63, 68, 5, 39, 0, 0, 64, 67, 3, 45, 22, 0, 65, 67, 9, 0, 0, 0, 66, 64,
		1, 0, 0, 0, 66, 65, 1, 0, 0, 0, 67, 70, 1, 0, 0, 0, 68, 69, 1, 0, 0, 0,
		68, 66, 1, 0, 0, 0, 69, 71, 1, 0, 0, 0, 70, 68, 1, 0, 0, 0, 71, 72, 5,
		39, 0, 0, 72, 8, 1, 0, 0, 0, 73, 74, 5, 116, 0, 0, 74, 75, 5, 114, 0, 0,
		75, 76, 5, 117, 0, 0, 76, 83, 5, 101, 0, 0, 77, 78, 5, 102, 0, 0, 78, 79,
		5, 97, 0, 0, 79, 80, 5, 108, 0, 0, 80, 81, 5, 115, 0, 0, 81, 83, 5, 101,
		0, 0, 82, 73, 1, 0, 0, 0, 82, 77, 1, 0, 0, 0, 83, 10, 1, 0, 0, 0, 84, 85,
		5, 110, 0, 0, 85, 86, 5, 117, 0, 0, 86, 87, 5, 108, 0, 0, 87, 88, 5, 108,
		0, 0, 88, 12, 1, 0, 0, 0, 89, 90, 5, 36, 0, 0, 90, 91, 5, 123, 0, 0, 91,
		92, 5, 123, 0, 0, 92, 14, 1, 0, 0, 0, 93, 94, 5, 125, 0, 0, 94, 95, 5,
		125, 0, 0, 95, 16, 1, 0, 0, 0, 96, 99, 3, 47, 23, 0, 97, 99, 3, 49, 24,
		0, 98, 96, 1, 0, 0, 0, 98, 97, 1, 0, 0, 0, 99, 18, 1, 0, 0, 0, 100, 101,
		5, 61, 0, 0, 101, 102, 5, 61, 0, 0, 102, 20, 1, 0, 0, 0, 103, 104, 5, 33,
		0, 0, 104, 105, 5, 61, 0, 0, 105, 22, 1, 0, 0, 0, 106, 107, 5, 62, 0, 0,
		107, 24, 1, 0, 0, 0, 108, 109, 5, 60, 0, 0, 109, 26, 1, 0, 0, 0, 110, 111,
		5, 62, 0, 0, 111, 112, 5, 61, 0, 0, 112, 28, 1, 0, 0, 0, 113, 114, 5, 60,
		0, 0, 114, 115, 5, 61, 0, 0, 115, 30, 1, 0, 0, 0, 116, 117, 3, 53, 26,
		0, 117, 32, 1, 0, 0, 0, 118, 119, 5, 40, 0, 0, 119, 34, 1, 0, 0, 0, 120,
		121, 5, 41, 0, 0, 121, 36, 1, 0, 0, 0, 122, 123, 5, 33, 0, 0, 123, 38,
		1, 0, 0, 0, 124, 125, 5, 124, 0, 0, 125, 126, 5, 124, 0, 0, 126, 40, 1,
		0, 0, 0, 127, 128, 5, 38, 0, 0, 128, 129, 5, 38, 0, 0, 129, 42, 1, 0, 0,
		0, 130, 131, 5, 46, 0, 0, 131, 44, 1, 0, 0, 0, 132, 133, 5, 92, 0, 0, 133,
		134, 7, 0, 0, 0, 134, 46, 1, 0, 0, 0, 135, 144, 5, 48, 0, 0, 136, 140,
		7, 1, 0, 0, 137, 139, 7, 2, 0, 0, 138, 137, 1, 0, 0, 0, 139, 142, 1, 0,
		0, 0, 140, 138, 1, 0, 0, 0, 140, 141, 1, 0, 0, 0, 141, 144, 1, 0, 0, 0,
		142, 140, 1, 0, 0, 0, 143, 135, 1, 0, 0, 0, 143, 136, 1, 0, 0, 0, 144,
		48, 1, 0, 0, 0, 145, 154, 5, 48, 0, 0, 146, 150, 7, 1, 0, 0, 147, 149,
		7, 2, 0, 0, 148, 147, 1, 0, 0, 0, 149, 152, 1, 0, 0, 0, 150, 148, 1, 0,
		0, 0, 150, 151, 1, 0, 0, 0, 151, 154, 1, 0, 0, 0, 152, 150, 1, 0, 0, 0,
		153, 145, 1, 0, 0, 0, 153, 146, 1, 0, 0, 0, 154, 155, 1, 0, 0, 0, 155,
		159, 5, 46, 0, 0, 156, 158, 7, 2, 0, 0, 157, 156, 1, 0, 0, 0, 158, 161,
		1, 0, 0, 0, 159, 157, 1, 0, 0, 0, 159, 160, 1, 0, 0, 0, 160, 163, 1, 0,
		0, 0, 161, 159, 1, 0, 0, 0, 162, 164, 3, 51, 25, 0, 163, 162, 1, 0, 0,
		0, 163, 164, 1, 0, 0, 0, 164, 186, 1, 0, 0, 0, 165, 167, 5, 46, 0, 0, 166,
		168, 7, 2, 0, 0, 167, 166, 1, 0, 0, 0, 168, 169, 1, 0, 0, 0, 169, 167,
		1, 0, 0, 0, 169, 170, 1, 0, 0, 0, 170, 172, 1, 0, 0, 0, 171, 173, 3, 51,
		25, 0, 172, 171, 1, 0, 0, 0, 172, 173, 1, 0, 0, 0, 173, 186, 1, 0, 0, 0,
		174, 183, 5, 48, 0, 0, 175, 179, 7, 1, 0, 0, 176, 178, 7, 2, 0, 0, 177,
		176, 1, 0, 0, 0, 178, 181, 1, 0, 0, 0, 179, 177, 1, 0, 0, 0, 179, 180,
		1, 0, 0, 0, 180, 183, 1, 0, 0, 0, 181, 179, 1, 0, 0, 0, 182, 174, 1, 0,
		0, 0, 182, 175, 1, 0, 0, 0, 183, 184, 1, 0, 0, 0, 184, 186, 3, 51, 25,
		0, 185, 153, 1, 0, 0, 0, 185, 165, 1, 0, 0, 0, 185, 182, 1, 0, 0, 0, 186,
		50, 1, 0, 0, 0, 187, 189, 7, 3, 0, 0, 188, 190, 7, 4, 0, 0, 189, 188, 1,
		0, 0, 0, 189, 190, 1, 0, 0, 0, 190, 192, 1, 0, 0, 0, 191, 193, 7, 2, 0,
		0, 192, 191, 1, 0, 0, 0, 193, 194, 1, 0, 0, 0, 194, 192, 1, 0, 0, 0, 194,
		195, 1, 0, 0, 0, 195, 52, 1, 0, 0, 0, 196, 200, 7, 5, 0, 0, 197, 199, 7,
		6, 0, 0, 198, 197, 1, 0, 0, 0, 199, 202, 1, 0, 0, 0, 200, 198, 1, 0, 0,
		0, 200, 201, 1, 0, 0, 0, 201, 54, 1, 0, 0, 0, 202, 200, 1, 0, 0, 0, 203,
		205, 7, 7, 0, 0, 204, 203, 1, 0, 0, 0, 205, 206, 1, 0, 0, 0, 206, 204,
		1, 0, 0, 0, 206, 207, 1, 0, 0, 0, 207, 208, 1, 0, 0, 0, 208, 209, 6, 27,
		0, 0, 209, 56, 1, 0, 0, 0, 20, 0, 66, 68, 82, 98, 140, 143, 150, 153, 159,
		163, 169, 172, 179, 182, 185, 189, 194, 200, 206, 1, 6, 0, 0,
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

// ActionLexerInit initializes any static state used to implement ActionLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewActionLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func ActionLexerInit() {
	staticData := &actionlexerLexerStaticData
	staticData.once.Do(actionlexerLexerInit)
}

// NewActionLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewActionLexer(input antlr.CharStream) *ActionLexer {
	ActionLexerInit()
	l := new(ActionLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &actionlexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.predictionContextCache)
	l.channelNames = staticData.channelNames
	l.modeNames = staticData.modeNames
	l.RuleNames = staticData.ruleNames
	l.LiteralNames = staticData.literalNames
	l.SymbolicNames = staticData.symbolicNames
	l.GrammarFileName = "Action.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// ActionLexer tokens.
const (
	ActionLexerT__0                     = 1
	ActionLexerT__1                     = 2
	ActionLexerT__2                     = 3
	ActionLexerSTRING_INSIDE_EXPRESSION = 4
	ActionLexerBOOLEAN                  = 5
	ActionLexerNULL                     = 6
	ActionLexerEXP_START                = 7
	ActionLexerEXP_END                  = 8
	ActionLexerNUMBER                   = 9
	ActionLexerEQ                       = 10
	ActionLexerNEQ                      = 11
	ActionLexerGT                       = 12
	ActionLexerLT                       = 13
	ActionLexerGTE                      = 14
	ActionLexerLTE                      = 15
	ActionLexerID                       = 16
	ActionLexerLPAREN                   = 17
	ActionLexerRPAREN                   = 18
	ActionLexerNOT                      = 19
	ActionLexerOR                       = 20
	ActionLexerAND                      = 21
	ActionLexerDOT                      = 22
	ActionLexerWS                       = 23
)
