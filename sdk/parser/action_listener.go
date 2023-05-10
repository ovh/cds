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

	// EnterVariableIdentifier is called when entering the variableIdentifier production.
	EnterVariableIdentifier(c *VariableIdentifierContext)

	// EnterNumberExpression is called when entering the numberExpression production.
	EnterNumberExpression(c *NumberExpressionContext)

	// EnterStringExpression is called when entering the stringExpression production.
	EnterStringExpression(c *StringExpressionContext)

	// EnterTermExpression is called when entering the termExpression production.
	EnterTermExpression(c *TermExpressionContext)

	// EnterNotExpression is called when entering the notExpression production.
	EnterNotExpression(c *NotExpressionContext)

	// EnterNotOperator is called when entering the notOperator production.
	EnterNotOperator(c *NotOperatorContext)

	// EnterFunctionCall is called when entering the functionCall production.
	EnterFunctionCall(c *FunctionCallContext)

	// EnterFunctionName is called when entering the functionName production.
	EnterFunctionName(c *FunctionNameContext)

	// EnterFunctionCallArguments is called when entering the functionCallArguments production.
	EnterFunctionCallArguments(c *FunctionCallArgumentsContext)

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

	// EnterBooleanExpression is called when entering the booleanExpression production.
	EnterBooleanExpression(c *BooleanExpressionContext)

	// EnterExpressionStart is called when entering the expressionStart production.
	EnterExpressionStart(c *ExpressionStartContext)

	// EnterExpressionEnd is called when entering the expressionEnd production.
	EnterExpressionEnd(c *ExpressionEndContext)

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

	// ExitVariableIdentifier is called when exiting the variableIdentifier production.
	ExitVariableIdentifier(c *VariableIdentifierContext)

	// ExitNumberExpression is called when exiting the numberExpression production.
	ExitNumberExpression(c *NumberExpressionContext)

	// ExitStringExpression is called when exiting the stringExpression production.
	ExitStringExpression(c *StringExpressionContext)

	// ExitTermExpression is called when exiting the termExpression production.
	ExitTermExpression(c *TermExpressionContext)

	// ExitNotExpression is called when exiting the notExpression production.
	ExitNotExpression(c *NotExpressionContext)

	// ExitNotOperator is called when exiting the notOperator production.
	ExitNotOperator(c *NotOperatorContext)

	// ExitFunctionCall is called when exiting the functionCall production.
	ExitFunctionCall(c *FunctionCallContext)

	// ExitFunctionName is called when exiting the functionName production.
	ExitFunctionName(c *FunctionNameContext)

	// ExitFunctionCallArguments is called when exiting the functionCallArguments production.
	ExitFunctionCallArguments(c *FunctionCallArgumentsContext)

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

	// ExitBooleanExpression is called when exiting the booleanExpression production.
	ExitBooleanExpression(c *BooleanExpressionContext)

	// ExitExpressionStart is called when exiting the expressionStart production.
	ExitExpressionStart(c *ExpressionStartContext)

	// ExitExpressionEnd is called when exiting the expressionEnd production.
	ExitExpressionEnd(c *ExpressionEndContext)
}
