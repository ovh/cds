// Code generated from Action.g4 by ANTLR 4.12.0. DO NOT EDIT.

package parser // Action

import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

// BaseActionListener is a complete listener for a parse tree produced by ActionParser.
type BaseActionListener struct{}

var _ ActionListener = &BaseActionListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseActionListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseActionListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseActionListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseActionListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterStart is called when production start is entered.
func (s *BaseActionListener) EnterStart(ctx *StartContext) {}

// ExitStart is called when production start is exited.
func (s *BaseActionListener) ExitStart(ctx *StartContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseActionListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseActionListener) ExitExpression(ctx *ExpressionContext) {}

// EnterOrExpression is called when production orExpression is entered.
func (s *BaseActionListener) EnterOrExpression(ctx *OrExpressionContext) {}

// ExitOrExpression is called when production orExpression is exited.
func (s *BaseActionListener) ExitOrExpression(ctx *OrExpressionContext) {}

// EnterAndExpression is called when production andExpression is entered.
func (s *BaseActionListener) EnterAndExpression(ctx *AndExpressionContext) {}

// ExitAndExpression is called when production andExpression is exited.
func (s *BaseActionListener) ExitAndExpression(ctx *AndExpressionContext) {}

// EnterComparisonExpression is called when production comparisonExpression is entered.
func (s *BaseActionListener) EnterComparisonExpression(ctx *ComparisonExpressionContext) {}

// ExitComparisonExpression is called when production comparisonExpression is exited.
func (s *BaseActionListener) ExitComparisonExpression(ctx *ComparisonExpressionContext) {}

// EnterEqualityExpression is called when production equalityExpression is entered.
func (s *BaseActionListener) EnterEqualityExpression(ctx *EqualityExpressionContext) {}

// ExitEqualityExpression is called when production equalityExpression is exited.
func (s *BaseActionListener) ExitEqualityExpression(ctx *EqualityExpressionContext) {}

// EnterPrimaryExpression is called when production primaryExpression is entered.
func (s *BaseActionListener) EnterPrimaryExpression(ctx *PrimaryExpressionContext) {}

// ExitPrimaryExpression is called when production primaryExpression is exited.
func (s *BaseActionListener) ExitPrimaryExpression(ctx *PrimaryExpressionContext) {}

// EnterVariableContext is called when production variableContext is entered.
func (s *BaseActionListener) EnterVariableContext(ctx *VariableContextContext) {}

// ExitVariableContext is called when production variableContext is exited.
func (s *BaseActionListener) ExitVariableContext(ctx *VariableContextContext) {}

// EnterVariablePath is called when production variablePath is entered.
func (s *BaseActionListener) EnterVariablePath(ctx *VariablePathContext) {}

// ExitVariablePath is called when production variablePath is exited.
func (s *BaseActionListener) ExitVariablePath(ctx *VariablePathContext) {}

// EnterVariableIdentifier is called when production variableIdentifier is entered.
func (s *BaseActionListener) EnterVariableIdentifier(ctx *VariableIdentifierContext) {}

// ExitVariableIdentifier is called when production variableIdentifier is exited.
func (s *BaseActionListener) ExitVariableIdentifier(ctx *VariableIdentifierContext) {}

// EnterNumberExpression is called when production numberExpression is entered.
func (s *BaseActionListener) EnterNumberExpression(ctx *NumberExpressionContext) {}

// ExitNumberExpression is called when production numberExpression is exited.
func (s *BaseActionListener) ExitNumberExpression(ctx *NumberExpressionContext) {}

// EnterStringExpression is called when production stringExpression is entered.
func (s *BaseActionListener) EnterStringExpression(ctx *StringExpressionContext) {}

// ExitStringExpression is called when production stringExpression is exited.
func (s *BaseActionListener) ExitStringExpression(ctx *StringExpressionContext) {}

// EnterTermExpression is called when production termExpression is entered.
func (s *BaseActionListener) EnterTermExpression(ctx *TermExpressionContext) {}

// ExitTermExpression is called when production termExpression is exited.
func (s *BaseActionListener) ExitTermExpression(ctx *TermExpressionContext) {}

// EnterNotExpression is called when production notExpression is entered.
func (s *BaseActionListener) EnterNotExpression(ctx *NotExpressionContext) {}

// ExitNotExpression is called when production notExpression is exited.
func (s *BaseActionListener) ExitNotExpression(ctx *NotExpressionContext) {}

// EnterNotOperator is called when production notOperator is entered.
func (s *BaseActionListener) EnterNotOperator(ctx *NotOperatorContext) {}

// ExitNotOperator is called when production notOperator is exited.
func (s *BaseActionListener) ExitNotOperator(ctx *NotOperatorContext) {}

// EnterFunctionCall is called when production functionCall is entered.
func (s *BaseActionListener) EnterFunctionCall(ctx *FunctionCallContext) {}

// ExitFunctionCall is called when production functionCall is exited.
func (s *BaseActionListener) ExitFunctionCall(ctx *FunctionCallContext) {}

// EnterFunctionName is called when production functionName is entered.
func (s *BaseActionListener) EnterFunctionName(ctx *FunctionNameContext) {}

// ExitFunctionName is called when production functionName is exited.
func (s *BaseActionListener) ExitFunctionName(ctx *FunctionNameContext) {}

// EnterFunctionCallArguments is called when production functionCallArguments is entered.
func (s *BaseActionListener) EnterFunctionCallArguments(ctx *FunctionCallArgumentsContext) {}

// ExitFunctionCallArguments is called when production functionCallArguments is exited.
func (s *BaseActionListener) ExitFunctionCallArguments(ctx *FunctionCallArgumentsContext) {}

// EnterArray is called when production array is entered.
func (s *BaseActionListener) EnterArray(ctx *ArrayContext) {}

// ExitArray is called when production array is exited.
func (s *BaseActionListener) ExitArray(ctx *ArrayContext) {}

// EnterArrayIndex is called when production arrayIndex is entered.
func (s *BaseActionListener) EnterArrayIndex(ctx *ArrayIndexContext) {}

// ExitArrayIndex is called when production arrayIndex is exited.
func (s *BaseActionListener) ExitArrayIndex(ctx *ArrayIndexContext) {}

// EnterAndOperator is called when production andOperator is entered.
func (s *BaseActionListener) EnterAndOperator(ctx *AndOperatorContext) {}

// ExitAndOperator is called when production andOperator is exited.
func (s *BaseActionListener) ExitAndOperator(ctx *AndOperatorContext) {}

// EnterOrOperator is called when production orOperator is entered.
func (s *BaseActionListener) EnterOrOperator(ctx *OrOperatorContext) {}

// ExitOrOperator is called when production orOperator is exited.
func (s *BaseActionListener) ExitOrOperator(ctx *OrOperatorContext) {}

// EnterComparisonOperator is called when production comparisonOperator is entered.
func (s *BaseActionListener) EnterComparisonOperator(ctx *ComparisonOperatorContext) {}

// ExitComparisonOperator is called when production comparisonOperator is exited.
func (s *BaseActionListener) ExitComparisonOperator(ctx *ComparisonOperatorContext) {}

// EnterEqualityOperator is called when production equalityOperator is entered.
func (s *BaseActionListener) EnterEqualityOperator(ctx *EqualityOperatorContext) {}

// ExitEqualityOperator is called when production equalityOperator is exited.
func (s *BaseActionListener) ExitEqualityOperator(ctx *EqualityOperatorContext) {}

// EnterBooleanExpression is called when production booleanExpression is entered.
func (s *BaseActionListener) EnterBooleanExpression(ctx *BooleanExpressionContext) {}

// ExitBooleanExpression is called when production booleanExpression is exited.
func (s *BaseActionListener) ExitBooleanExpression(ctx *BooleanExpressionContext) {}

// EnterExpressionStart is called when production expressionStart is entered.
func (s *BaseActionListener) EnterExpressionStart(ctx *ExpressionStartContext) {}

// ExitExpressionStart is called when production expressionStart is exited.
func (s *BaseActionListener) ExitExpressionStart(ctx *ExpressionStartContext) {}

// EnterExpressionEnd is called when production expressionEnd is entered.
func (s *BaseActionListener) EnterExpressionEnd(ctx *ExpressionEndContext) {}

// ExitExpressionEnd is called when production expressionEnd is exited.
func (s *BaseActionListener) ExitExpressionEnd(ctx *ExpressionEndContext) {}
