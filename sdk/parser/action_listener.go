// Code generated from Action.g4 by ANTLR 4.12.0. DO NOT EDIT.

package parser // Action

import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

// ActionListener is a complete listener for a parse tree produced by ActionParser.
type ActionListener interface {
	antlr.ParseTreeListener

	// EnterStart is called when entering the start production.
	EnterStart(c *StartContext)

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterOrExpression is called when entering the orExpression production.
	EnterOrExpression(c *OrExpressionContext)

	// EnterAndExpression is called when entering the andExpression production.
	EnterAndExpression(c *AndExpressionContext)

	// EnterComparisonExpression is called when entering the comparisonExpression production.
	EnterComparisonExpression(c *ComparisonExpressionContext)

	// EnterEqualityExpression is called when entering the equalityExpression production.
	EnterEqualityExpression(c *EqualityExpressionContext)

	// EnterPrimaryExpression is called when entering the primaryExpression production.
	EnterPrimaryExpression(c *PrimaryExpressionContext)

	// EnterVariableContext is called when entering the variableContext production.
	EnterVariableContext(c *VariableContextContext)

	// EnterVariablePath is called when entering the variablePath production.
	EnterVariablePath(c *VariablePathContext)

	// EnterNumberExpression is called when entering the numberExpression production.
	EnterNumberExpression(c *NumberExpressionContext)

	// EnterStringExpression is called when entering the stringExpression production.
	EnterStringExpression(c *StringExpressionContext)

	// EnterTermExpression is called when entering the termExpression production.
	EnterTermExpression(c *TermExpressionContext)

	// EnterNotExpression is called when entering the notExpression production.
	EnterNotExpression(c *NotExpressionContext)

	// EnterFunctionCall is called when entering the functionCall production.
	EnterFunctionCall(c *FunctionCallContext)

	// EnterFunctionCallArguments is called when entering the functionCallArguments production.
	EnterFunctionCallArguments(c *FunctionCallArgumentsContext)

	// EnterFunctionCallArg is called when entering the functionCallArg production.
	EnterFunctionCallArg(c *FunctionCallArgContext)

	// EnterArray is called when entering the array production.
	EnterArray(c *ArrayContext)

	// EnterArrayIndex is called when entering the arrayIndex production.
	EnterArrayIndex(c *ArrayIndexContext)

	// EnterAndOperator is called when entering the andOperator production.
	EnterAndOperator(c *AndOperatorContext)

	// EnterOrOperator is called when entering the orOperator production.
	EnterOrOperator(c *OrOperatorContext)

	// EnterComparisonOperator is called when entering the comparisonOperator production.
	EnterComparisonOperator(c *ComparisonOperatorContext)

	// EnterEqualityOperator is called when entering the equalityOperator production.
	EnterEqualityOperator(c *EqualityOperatorContext)

	// EnterLiteral is called when entering the literal production.
	EnterLiteral(c *LiteralContext)

	// ExitStart is called when exiting the start production.
	ExitStart(c *StartContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitOrExpression is called when exiting the orExpression production.
	ExitOrExpression(c *OrExpressionContext)

	// ExitAndExpression is called when exiting the andExpression production.
	ExitAndExpression(c *AndExpressionContext)

	// ExitComparisonExpression is called when exiting the comparisonExpression production.
	ExitComparisonExpression(c *ComparisonExpressionContext)

	// ExitEqualityExpression is called when exiting the equalityExpression production.
	ExitEqualityExpression(c *EqualityExpressionContext)

	// ExitPrimaryExpression is called when exiting the primaryExpression production.
	ExitPrimaryExpression(c *PrimaryExpressionContext)

	// ExitVariableContext is called when exiting the variableContext production.
	ExitVariableContext(c *VariableContextContext)

	// ExitVariablePath is called when exiting the variablePath production.
	ExitVariablePath(c *VariablePathContext)

	// ExitNumberExpression is called when exiting the numberExpression production.
	ExitNumberExpression(c *NumberExpressionContext)

	// ExitStringExpression is called when exiting the stringExpression production.
	ExitStringExpression(c *StringExpressionContext)

	// ExitTermExpression is called when exiting the termExpression production.
	ExitTermExpression(c *TermExpressionContext)

	// ExitNotExpression is called when exiting the notExpression production.
	ExitNotExpression(c *NotExpressionContext)

	// ExitFunctionCall is called when exiting the functionCall production.
	ExitFunctionCall(c *FunctionCallContext)

	// ExitFunctionCallArguments is called when exiting the functionCallArguments production.
	ExitFunctionCallArguments(c *FunctionCallArgumentsContext)

	// ExitFunctionCallArg is called when exiting the functionCallArg production.
	ExitFunctionCallArg(c *FunctionCallArgContext)

	// ExitArray is called when exiting the array production.
	ExitArray(c *ArrayContext)

	// ExitArrayIndex is called when exiting the arrayIndex production.
	ExitArrayIndex(c *ArrayIndexContext)

	// ExitAndOperator is called when exiting the andOperator production.
	ExitAndOperator(c *AndOperatorContext)

	// ExitOrOperator is called when exiting the orOperator production.
	ExitOrOperator(c *OrOperatorContext)

	// ExitComparisonOperator is called when exiting the comparisonOperator production.
	ExitComparisonOperator(c *ComparisonOperatorContext)

	// ExitEqualityOperator is called when exiting the equalityOperator production.
	ExitEqualityOperator(c *EqualityOperatorContext)

	// ExitLiteral is called when exiting the literal production.
	ExitLiteral(c *LiteralContext)
}