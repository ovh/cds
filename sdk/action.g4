grammar action;


start: line;
line: (nonexpression | expressionStart)* EOF;
expressionStart: EXP_START expression (expression)* EXP_END;
expression: orExpression;
orExpression: andExpression ('||' andExpression)*;
andExpression: comparisonExpression ('&&' comparisonExpression)*;
comparisonExpression: equalityExpression (comparisonOperator equalityExpression)*;
equalityExpression: primaryExpression (equalityOperator primaryExpression)*;
primaryExpression: variableReference | NUMBER | (functionCall ('.' variableReference)*) | STRING_INSIDE_EXPRESSION;
variableReference: ID ( ( '.' ID )* | array);
functionCall: ID LPAREN functionCallArguments RPAREN;
functionCallArguments: functionCallArg (',' functionCallArg)*;
functionCallArg
    : // No arguments
    | primaryExpression* // Some arguments
    | literal
    ;
array: '[' arrayIndex ']';
arrayIndex: NUMBER | expression;
comparisonOperator: (GT | LT | GTE | LTE);
equalityOperator: (EQ | NEQ);
literal: STRING_INSIDE_EXPRESSION | BOOLEAN | NULL | NUMBER;
nonexpression: CHAR;

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
POW         : '^';
ID          : IDENTIFIER;
LPAREN      : '(';
RPAREN      : ')';
CHAR: [a-zA-Z_0-9-].*;

fragment ESC: '\\' ["'\\/bfnrt];
fragment INT: ('0' | [1-9][0-9]*) ;
fragment FLOAT: ('0'|[1-9][0-9]*) '.' [0-9]* EXPONENT? | '.' [0-9]+ EXPONENT? | ('0'|[1-9][0-9]*) EXPONENT;
fragment EXPONENT: [Ee] [+\-]? [0-9]+ ;
fragment IDENTIFIER: [a-zA-Z_] [a-zA-Z_0-9-]*;
WS: [ \t\r\n]+ -> skip;

//${{ github.event_name }}
//${{ github.ref }}
//${{ steps.my_step.outputs.my_output }}
//${{ format('Hello {0}!', 'world') }}
//${{ env.MY_VARIABLE }}
//${{ always() }}
//${{ contains('Hello world', 'world') }}
// ${{ toInt('42') }}
// ${{ github.repository }}
//(${{ matrix.foo }} == 'bar' && ${{ env.BAZ }} != 'qux') || (${{ github.event_name }} == 'pull_request' && ${{ github.event.pull_request.head.ref }})} == 'main')
//echo "Hello world" && [ ${{ steps.myStep.outputs.result }} = "success" ] && echo "Step completed successfully"
// ${{ fromJson(steps.myStep.outputs.response).data[0].name }}
//(${{ success() }} || ${{ failure() }}) && ${{ always() }}
//if [ "${{ matrix.os }}" = "ubuntu-latest" ] || [ "${{ matrix.os }}" = "macos-latest" ]; then echo "Hello, Linux/MacOS!"; fi
//if [ "${{ contains(github.event.head_commit.message, 'Fix') }}" = "true" ]; then echo "Commit message contains 'Fix'"; fi
//run: docker build -t my-image:${{ github.sha }} . && docker push my-image:${{ github.sha }}
//if [ "${{ github.ref == 'refs/heads/main' && github.event_name == 'push' }}" = "true" ]; then echo "Push to main branch!"; fi
// echo "The selected color is ${{ steps.random-color-generator.outputs.SELECTED_COLOR }}"
// echo "SELECTED_COLOR=green" >> "$GITHUB_OUTPUT"
