grammar Action;


start: expression EOF;
expression: (expressionStart orExpression (orExpression)* expressionEnd);
orExpression: andExpression (orOperator andExpression)*;
andExpression: comparisonExpression (andOperator comparisonExpression)*;
comparisonExpression: equalityExpression (comparisonOperator equalityExpression)?;
equalityExpression: primaryExpression (equalityOperator primaryExpression)?;
primaryExpression: variableContext | numberExpression | functionCall | stringExpression | termExpression | notExpression;
variableContext: variableIdentifier variablePath*;
variablePath: (DOT variableIdentifier | array | DOT filterExpression);
variableIdentifier: ID;
numberExpression: NUMBER;
stringExpression: STRING_INSIDE_EXPRESSION;
termExpression: LPAREN orExpression RPAREN;
notExpression: (notOperator primaryExpression);
notOperator: NOT;
functionCall: functionName LPAREN functionCallArguments (',' functionCallArguments)* RPAREN;
functionName: ID;
functionCallArguments
    : // No arguments
    | variableContext
    | stringExpression
    | numberExpression
    | booleanExpression

    ;
array: '[' arrayIndex ']';
arrayIndex: primaryExpression;
andOperator: AND;
orOperator: OR;
comparisonOperator: (GT | LT | GTE | LTE);
equalityOperator: (EQ | NEQ);
booleanExpression: BOOLEAN;
expressionStart: EXP_START;
expressionEnd: EXP_END;
filterExpression: STAR;

STRING_INSIDE_EXPRESSION: '\'' (ESC|.)*? '\'';
BOOLEAN: 'true' | 'false';
NULL: 'null';
EXP_START: '${{';
EXP_END: '}}';
NUMBER: INT | FLOAT;
EQ          : '==' ;
NEQ         : '!=' ;
GT          : '>' ;
LT          : '<' ;
GTE         : '>=' ;
LTE         : '<=' ;
ID          : IDENTIFIER;
LPAREN      : '(';
RPAREN      : ')';
NOT         : '!';
OR          : '||';
AND         : '&&';
DOT         : '.';
STAR        : '*';

fragment ESC: '\\' ["'\\/bfnrt];
fragment INT: ('0' | [1-9][0-9]*) ;
fragment FLOAT: ('0'|[1-9][0-9]*) '.' [0-9]* EXPONENT? | '.' [0-9]+ EXPONENT? | ('0'|[1-9][0-9]*) EXPONENT;
fragment EXPONENT: [Ee] [+\-]? [0-9]+ ;
fragment IDENTIFIER: [a-zA-Z_] [a-zA-Z_0-9-]*;

WS: [ \t\r\n]+ -> skip;
