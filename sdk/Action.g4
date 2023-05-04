grammar Action;


start: expression EOF;
expression: (EXP_START orExpression (orExpression)* EXP_END);
orExpression: andExpression (orOperator andExpression)*;
andExpression: comparisonExpression (andOperator comparisonExpression)*;
comparisonExpression: equalityExpression (comparisonOperator equalityExpression)*;
equalityExpression: primaryExpression (equalityOperator primaryExpression)*;
primaryExpression: variableContext | numberExpression | functionCall | stringExpression | termExpression | notExpression;
variableContext: ID variablePath*;
variablePath: ('.' ID | array) ;
numberExpression: NUMBER;
stringExpression: STRING_INSIDE_EXPRESSION;
termExpression: '(' orExpression ')';
notExpression: (NOT primaryExpression);
functionCall: ID LPAREN functionCallArguments RPAREN (variablePath)*;
functionCallArguments: functionCallArg (',' functionCallArg)*;
functionCallArg
    : // No arguments
    | primaryExpression* // Some arguments
    | literal
    ;
array: '[' arrayIndex ']';
arrayIndex: primaryExpression;
andOperator: AND;
orOperator: OR;
comparisonOperator: (GT | LT | GTE | LTE);
equalityOperator: (EQ | NEQ);
literal: STRING_INSIDE_EXPRESSION | BOOLEAN | NULL | NUMBER;

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

fragment ESC: '\\' ["'\\/bfnrt];
fragment INT: ('0' | [1-9][0-9]*) ;
fragment FLOAT: ('0'|[1-9][0-9]*) '.' [0-9]* EXPONENT? | '.' [0-9]+ EXPONENT? | ('0'|[1-9][0-9]*) EXPONENT;
fragment EXPONENT: [Ee] [+\-]? [0-9]+ ;
fragment IDENTIFIER: [a-zA-Z_] [a-zA-Z_0-9-]*;

WS: [ \t\r\n]+ -> skip;

// ${{ github.repository }}
//(${{ matrix.foo }} == 'bar' && ${{ env.BAZ }} != 'qux') || (${{ github.event_name }} == 'pull_request' && ${{ github.event.pull_request.head.ref }})} == 'main')

//if [ "${{ matrix.os }}" = "ubuntu-latest" ] || [ "${{ matrix.os }}" = "macos-latest" ]; then echo "Hello, Linux/MacOS!"; fi
//if [ "${{ contains(github.event.head_commit.message, 'Fix') }}" = "true" ]; then echo "Commit message contains 'Fix'"; fi

//docker build -t my-image:${{ github.sha }} . && docker push my-image:${{ github.sha }}
//if [ "${{ github.ref == 'refs/heads/main' && github.event_name == 'push' }}" = "true" ]; then echo "Push to main branch!"; fi

// echo "The selected color is ${{ steps.random-color-generator.outputs.SELECTED_COLOR }}"
// echo "SELECTED_COLOR=green" >> "$GITHUB_OUTPUT"
// ${{ input.data }}
// ${{ toJSON(input.data).data[0].num }}
// ${{ input.data.data[0].num }}
// ${{ toInt('42') }}
// ${{ format('Hello {0}!', 'world') }}