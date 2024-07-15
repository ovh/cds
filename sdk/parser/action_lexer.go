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
	modeNames []string
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
    "'&&'", "'.'", "'*'",
  }
  staticData.symbolicNames = []string{
    "", "", "", "", "STRING_INSIDE_EXPRESSION", "BOOLEAN", "NULL", "EXP_START", 
    "EXP_END", "NUMBER", "EQ", "NEQ", "GT", "LT", "GTE", "LTE", "ID", "LPAREN", 
    "RPAREN", "NOT", "OR", "AND", "DOT", "STAR", "WS",
  }
  staticData.ruleNames = []string{
    "T__0", "T__1", "T__2", "STRING_INSIDE_EXPRESSION", "BOOLEAN", "NULL", 
    "EXP_START", "EXP_END", "NUMBER", "EQ", "NEQ", "GT", "LT", "GTE", "LTE", 
    "ID", "LPAREN", "RPAREN", "NOT", "OR", "AND", "DOT", "STAR", "ESC", 
    "INT", "FLOAT", "EXPONENT", "IDENTIFIER", "WS",
  }
  staticData.predictionContextCache = antlr.NewPredictionContextCache()
  staticData.serializedATN = []int32{
	4, 0, 24, 214, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 
	4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 
	10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 
	7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 
	20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 
	2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 
	1, 2, 1, 3, 1, 3, 1, 3, 5, 3, 69, 8, 3, 10, 3, 12, 3, 72, 9, 3, 1, 3, 1, 
	3, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 3, 4, 85, 8, 4, 
	1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 
	1, 8, 1, 8, 3, 8, 101, 8, 8, 1, 9, 1, 9, 1, 9, 1, 10, 1, 10, 1, 10, 1, 
	11, 1, 11, 1, 12, 1, 12, 1, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 14, 1, 15, 
	1, 15, 1, 16, 1, 16, 1, 17, 1, 17, 1, 18, 1, 18, 1, 19, 1, 19, 1, 19, 1, 
	20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 22, 1, 22, 1, 23, 1, 23, 1, 23, 1, 24, 
	1, 24, 1, 24, 5, 24, 143, 8, 24, 10, 24, 12, 24, 146, 9, 24, 3, 24, 148, 
	8, 24, 1, 25, 1, 25, 1, 25, 5, 25, 153, 8, 25, 10, 25, 12, 25, 156, 9, 
	25, 3, 25, 158, 8, 25, 1, 25, 1, 25, 5, 25, 162, 8, 25, 10, 25, 12, 25, 
	165, 9, 25, 1, 25, 3, 25, 168, 8, 25, 1, 25, 1, 25, 4, 25, 172, 8, 25, 
	11, 25, 12, 25, 173, 1, 25, 3, 25, 177, 8, 25, 1, 25, 1, 25, 1, 25, 5, 
	25, 182, 8, 25, 10, 25, 12, 25, 185, 9, 25, 3, 25, 187, 8, 25, 1, 25, 3, 
	25, 190, 8, 25, 1, 26, 1, 26, 3, 26, 194, 8, 26, 1, 26, 4, 26, 197, 8, 
	26, 11, 26, 12, 26, 198, 1, 27, 1, 27, 5, 27, 203, 8, 27, 10, 27, 12, 27, 
	206, 9, 27, 1, 28, 4, 28, 209, 8, 28, 11, 28, 12, 28, 210, 1, 28, 1, 28, 
	1, 70, 0, 29, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 
	19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 
	37, 19, 39, 20, 41, 21, 43, 22, 45, 23, 47, 0, 49, 0, 51, 0, 53, 0, 55, 
	0, 57, 24, 1, 0, 8, 9, 0, 34, 34, 39, 39, 47, 47, 92, 92, 98, 98, 102, 
	102, 110, 110, 114, 114, 116, 116, 1, 0, 49, 57, 1, 0, 48, 57, 2, 0, 69, 
	69, 101, 101, 2, 0, 43, 43, 45, 45, 3, 0, 65, 90, 95, 95, 97, 122, 5, 0, 
	45, 45, 48, 57, 65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13, 32, 32, 228, 
	0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 
	0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 
	0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 
	0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 
	0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 
	1, 0, 0, 0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0, 0, 0, 0, 45, 1, 0, 0, 0, 0, 
	57, 1, 0, 0, 0, 1, 59, 1, 0, 0, 0, 3, 61, 1, 0, 0, 0, 5, 63, 1, 0, 0, 0, 
	7, 65, 1, 0, 0, 0, 9, 84, 1, 0, 0, 0, 11, 86, 1, 0, 0, 0, 13, 91, 1, 0, 
	0, 0, 15, 95, 1, 0, 0, 0, 17, 100, 1, 0, 0, 0, 19, 102, 1, 0, 0, 0, 21, 
	105, 1, 0, 0, 0, 23, 108, 1, 0, 0, 0, 25, 110, 1, 0, 0, 0, 27, 112, 1, 
	0, 0, 0, 29, 115, 1, 0, 0, 0, 31, 118, 1, 0, 0, 0, 33, 120, 1, 0, 0, 0, 
	35, 122, 1, 0, 0, 0, 37, 124, 1, 0, 0, 0, 39, 126, 1, 0, 0, 0, 41, 129, 
	1, 0, 0, 0, 43, 132, 1, 0, 0, 0, 45, 134, 1, 0, 0, 0, 47, 136, 1, 0, 0, 
	0, 49, 147, 1, 0, 0, 0, 51, 189, 1, 0, 0, 0, 53, 191, 1, 0, 0, 0, 55, 200, 
	1, 0, 0, 0, 57, 208, 1, 0, 0, 0, 59, 60, 5, 44, 0, 0, 60, 2, 1, 0, 0, 0, 
	61, 62, 5, 91, 0, 0, 62, 4, 1, 0, 0, 0, 63, 64, 5, 93, 0, 0, 64, 6, 1, 
	0, 0, 0, 65, 70, 5, 39, 0, 0, 66, 69, 3, 47, 23, 0, 67, 69, 9, 0, 0, 0, 
	68, 66, 1, 0, 0, 0, 68, 67, 1, 0, 0, 0, 69, 72, 1, 0, 0, 0, 70, 71, 1, 
	0, 0, 0, 70, 68, 1, 0, 0, 0, 71, 73, 1, 0, 0, 0, 72, 70, 1, 0, 0, 0, 73, 
	74, 5, 39, 0, 0, 74, 8, 1, 0, 0, 0, 75, 76, 5, 116, 0, 0, 76, 77, 5, 114, 
	0, 0, 77, 78, 5, 117, 0, 0, 78, 85, 5, 101, 0, 0, 79, 80, 5, 102, 0, 0, 
	80, 81, 5, 97, 0, 0, 81, 82, 5, 108, 0, 0, 82, 83, 5, 115, 0, 0, 83, 85, 
	5, 101, 0, 0, 84, 75, 1, 0, 0, 0, 84, 79, 1, 0, 0, 0, 85, 10, 1, 0, 0, 
	0, 86, 87, 5, 110, 0, 0, 87, 88, 5, 117, 0, 0, 88, 89, 5, 108, 0, 0, 89, 
	90, 5, 108, 0, 0, 90, 12, 1, 0, 0, 0, 91, 92, 5, 36, 0, 0, 92, 93, 5, 123, 
	0, 0, 93, 94, 5, 123, 0, 0, 94, 14, 1, 0, 0, 0, 95, 96, 5, 125, 0, 0, 96, 
	97, 5, 125, 0, 0, 97, 16, 1, 0, 0, 0, 98, 101, 3, 49, 24, 0, 99, 101, 3, 
	51, 25, 0, 100, 98, 1, 0, 0, 0, 100, 99, 1, 0, 0, 0, 101, 18, 1, 0, 0, 
	0, 102, 103, 5, 61, 0, 0, 103, 104, 5, 61, 0, 0, 104, 20, 1, 0, 0, 0, 105, 
	106, 5, 33, 0, 0, 106, 107, 5, 61, 0, 0, 107, 22, 1, 0, 0, 0, 108, 109, 
	5, 62, 0, 0, 109, 24, 1, 0, 0, 0, 110, 111, 5, 60, 0, 0, 111, 26, 1, 0, 
	0, 0, 112, 113, 5, 62, 0, 0, 113, 114, 5, 61, 0, 0, 114, 28, 1, 0, 0, 0, 
	115, 116, 5, 60, 0, 0, 116, 117, 5, 61, 0, 0, 117, 30, 1, 0, 0, 0, 118, 
	119, 3, 55, 27, 0, 119, 32, 1, 0, 0, 0, 120, 121, 5, 40, 0, 0, 121, 34, 
	1, 0, 0, 0, 122, 123, 5, 41, 0, 0, 123, 36, 1, 0, 0, 0, 124, 125, 5, 33, 
	0, 0, 125, 38, 1, 0, 0, 0, 126, 127, 5, 124, 0, 0, 127, 128, 5, 124, 0, 
	0, 128, 40, 1, 0, 0, 0, 129, 130, 5, 38, 0, 0, 130, 131, 5, 38, 0, 0, 131, 
	42, 1, 0, 0, 0, 132, 133, 5, 46, 0, 0, 133, 44, 1, 0, 0, 0, 134, 135, 5, 
	42, 0, 0, 135, 46, 1, 0, 0, 0, 136, 137, 5, 92, 0, 0, 137, 138, 7, 0, 0, 
	0, 138, 48, 1, 0, 0, 0, 139, 148, 5, 48, 0, 0, 140, 144, 7, 1, 0, 0, 141, 
	143, 7, 2, 0, 0, 142, 141, 1, 0, 0, 0, 143, 146, 1, 0, 0, 0, 144, 142, 
	1, 0, 0, 0, 144, 145, 1, 0, 0, 0, 145, 148, 1, 0, 0, 0, 146, 144, 1, 0, 
	0, 0, 147, 139, 1, 0, 0, 0, 147, 140, 1, 0, 0, 0, 148, 50, 1, 0, 0, 0, 
	149, 158, 5, 48, 0, 0, 150, 154, 7, 1, 0, 0, 151, 153, 7, 2, 0, 0, 152, 
	151, 1, 0, 0, 0, 153, 156, 1, 0, 0, 0, 154, 152, 1, 0, 0, 0, 154, 155, 
	1, 0, 0, 0, 155, 158, 1, 0, 0, 0, 156, 154, 1, 0, 0, 0, 157, 149, 1, 0, 
	0, 0, 157, 150, 1, 0, 0, 0, 158, 159, 1, 0, 0, 0, 159, 163, 5, 46, 0, 0, 
	160, 162, 7, 2, 0, 0, 161, 160, 1, 0, 0, 0, 162, 165, 1, 0, 0, 0, 163, 
	161, 1, 0, 0, 0, 163, 164, 1, 0, 0, 0, 164, 167, 1, 0, 0, 0, 165, 163, 
	1, 0, 0, 0, 166, 168, 3, 53, 26, 0, 167, 166, 1, 0, 0, 0, 167, 168, 1, 
	0, 0, 0, 168, 190, 1, 0, 0, 0, 169, 171, 5, 46, 0, 0, 170, 172, 7, 2, 0, 
	0, 171, 170, 1, 0, 0, 0, 172, 173, 1, 0, 0, 0, 173, 171, 1, 0, 0, 0, 173, 
	174, 1, 0, 0, 0, 174, 176, 1, 0, 0, 0, 175, 177, 3, 53, 26, 0, 176, 175, 
	1, 0, 0, 0, 176, 177, 1, 0, 0, 0, 177, 190, 1, 0, 0, 0, 178, 187, 5, 48, 
	0, 0, 179, 183, 7, 1, 0, 0, 180, 182, 7, 2, 0, 0, 181, 180, 1, 0, 0, 0, 
	182, 185, 1, 0, 0, 0, 183, 181, 1, 0, 0, 0, 183, 184, 1, 0, 0, 0, 184, 
	187, 1, 0, 0, 0, 185, 183, 1, 0, 0, 0, 186, 178, 1, 0, 0, 0, 186, 179, 
	1, 0, 0, 0, 187, 188, 1, 0, 0, 0, 188, 190, 3, 53, 26, 0, 189, 157, 1, 
	0, 0, 0, 189, 169, 1, 0, 0, 0, 189, 186, 1, 0, 0, 0, 190, 52, 1, 0, 0, 
	0, 191, 193, 7, 3, 0, 0, 192, 194, 7, 4, 0, 0, 193, 192, 1, 0, 0, 0, 193, 
	194, 1, 0, 0, 0, 194, 196, 1, 0, 0, 0, 195, 197, 7, 2, 0, 0, 196, 195, 
	1, 0, 0, 0, 197, 198, 1, 0, 0, 0, 198, 196, 1, 0, 0, 0, 198, 199, 1, 0, 
	0, 0, 199, 54, 1, 0, 0, 0, 200, 204, 7, 5, 0, 0, 201, 203, 7, 6, 0, 0, 
	202, 201, 1, 0, 0, 0, 203, 206, 1, 0, 0, 0, 204, 202, 1, 0, 0, 0, 204, 
	205, 1, 0, 0, 0, 205, 56, 1, 0, 0, 0, 206, 204, 1, 0, 0, 0, 207, 209, 7, 
	7, 0, 0, 208, 207, 1, 0, 0, 0, 209, 210, 1, 0, 0, 0, 210, 208, 1, 0, 0, 
	0, 210, 211, 1, 0, 0, 0, 211, 212, 1, 0, 0, 0, 212, 213, 6, 28, 0, 0, 213, 
	58, 1, 0, 0, 0, 20, 0, 68, 70, 84, 100, 144, 147, 154, 157, 163, 167, 173, 
	176, 183, 186, 189, 193, 198, 204, 210, 1, 6, 0, 0,
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
	ActionLexerT__0 = 1
	ActionLexerT__1 = 2
	ActionLexerT__2 = 3
	ActionLexerSTRING_INSIDE_EXPRESSION = 4
	ActionLexerBOOLEAN = 5
	ActionLexerNULL = 6
	ActionLexerEXP_START = 7
	ActionLexerEXP_END = 8
	ActionLexerNUMBER = 9
	ActionLexerEQ = 10
	ActionLexerNEQ = 11
	ActionLexerGT = 12
	ActionLexerLT = 13
	ActionLexerGTE = 14
	ActionLexerLTE = 15
	ActionLexerID = 16
	ActionLexerLPAREN = 17
	ActionLexerRPAREN = 18
	ActionLexerNOT = 19
	ActionLexerOR = 20
	ActionLexerAND = 21
	ActionLexerDOT = 22
	ActionLexerSTAR = 23
	ActionLexerWS = 24
)

