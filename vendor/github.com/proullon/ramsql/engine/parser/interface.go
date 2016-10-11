package parser

import (
	"errors"
)

// ParseInstruction calls lexer and parser, then return Decl tree for each instruction
func ParseInstruction(instruction string) ([]Instruction, error) {

	l := lexer{}
	tokens, err := l.lex([]byte(instruction))
	if err != nil {
		return nil, err
	}

	p := parser{}
	instructions, err := p.parse(tokens)
	if err != nil {
		return nil, err
	}

	if len(instructions) == 0 {
		return nil, errors.New("Error in syntax near " + instruction)
	}

	return instructions, nil
}
