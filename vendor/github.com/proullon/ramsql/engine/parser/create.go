package parser

import (
	"fmt"
	"strings"
)

func (p *parser) parseCreate(tokens []Token) (*Instruction, error) {
	i := &Instruction{}

	// Set CREATE decl
	createDecl := NewDecl(tokens[p.index])
	i.Decls = append(i.Decls, createDecl)

	// After create token, should be either
	// TABLE
	// INDEX
	// ...
	if !p.hasNext() {
		return nil, fmt.Errorf("CREATE token must be followed by TABLE, INDEX")
	}
	p.index++

	switch tokens[p.index].Token {
	case TableToken:
		d, err := p.parseTable(tokens)
		if err != nil {
			return nil, err
		}
		createDecl.Add(d)
		break
	default:
		return nil, fmt.Errorf("Parsing error near <%s>", tokens[p.index].Lexeme)
	}

	return i, nil
}

func (p *parser) parseTable(tokens []Token) (*Decl, error) {
	var err error
	tableDecl := NewDecl(tokens[p.index])
	p.index++

	// Maybe have "IF NOT EXISTS" here
	if p.is(IfToken) {
		ifDecl, err := p.consumeToken(IfToken)
		if err != nil {
			return nil, err
		}
		tableDecl.Add(ifDecl)

		if p.is(NotToken) {
			notDecl, err := p.consumeToken(NotToken)
			if err != nil {
				return nil, err
			}
			ifDecl.Add(notDecl)
			if !p.is(ExistsToken) {
				return nil, p.syntaxError()
			}
			existsDecl, err := p.consumeToken(ExistsToken)
			if err != nil {
				return nil, err
			}
			notDecl.Add(existsDecl)
		}
	}

	// Now we should found table name
	nameTable, err := p.parseAttribute()
	if err != nil {
		return nil, p.syntaxError()
	}
	tableDecl.Add(nameTable)

	// Now we should found brackets
	if !p.hasNext() || tokens[p.index].Token != BracketOpeningToken {
		return nil, fmt.Errorf("Table name token must be followed by table definition")
	}
	p.index++

	for p.index < len(tokens) {

		switch p.cur().Token {
		case PrimaryToken:
			_, err := p.parsePrimaryKey()
			if err != nil {
				return nil, err
			}
			continue
		default:
		}

		// Closing bracket ?
		if tokens[p.index].Token == BracketClosingToken {
			p.consumeToken(BracketClosingToken)
			break
		}

		// New attribute name
		newAttribute, err := p.parseQuotedToken()
		if err != nil {
			return nil, err
		}
		tableDecl.Add(newAttribute)

		newAttributeType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		newAttribute.Add(newAttributeType)

		// Is it not null ?
		if _, err = p.isNext(NullToken); p.is(NotToken) && err == nil {
			notDecl, err := p.consumeToken(NotToken)
			if err != nil {
				return nil, err
			}
			newAttribute.Add(notDecl)
			nullDecl, err := p.consumeToken(NullToken)
			if err != nil {
				return nil, err
			}
			notDecl.Add(nullDecl)
		}

		// unique ?
		if tokens[p.index].Token == UniqueToken {
			p.consumeToken(UniqueToken)
		}

		// Is it a primary key ?
		if tokens[p.index].Token == PrimaryToken && p.hasNext() && tokens[p.index+1].Token == KeyToken {
			newPrimary := NewDecl(tokens[p.index])
			newAttribute.Add(newPrimary)

			if err = p.next(); err != nil {
				return nil, fmt.Errorf("Unexpected end")
			}

			newKey := NewDecl(tokens[p.index])
			newPrimary.Add(newKey)

			if err = p.next(); err != nil {
				return nil, fmt.Errorf("Unexpected end")
			}
		}

		// ANOTHER PROPERTY FFS ! Autoincrement ?
		if p.is(AutoincrementToken) {
			autoincDecl, err := p.consumeToken(AutoincrementToken)
			if err != nil {
				return nil, err
			}
			newAttribute.Add(autoincDecl)
		}

		// IF timestamp, maybe WITH TIME ZONE ?
		if strings.ToLower(newAttributeType.Lexeme) == "timestamp" && p.cur().Token == WithToken {
			withDecl, err := p.consumeToken(WithToken)
			if err != nil {
				return nil, err
			}
			timeDecl, err := p.consumeToken(TimeToken)
			if err != nil {
				return nil, err
			}
			zoneDecl, err := p.consumeToken(ZoneToken)
			if err != nil {
				return nil, err
			}
			newAttributeType.Add(withDecl)
			withDecl.Add(timeDecl)
			timeDecl.Add(zoneDecl)
		}

		// is it default ?
		if p.is(DefaultToken) {
			dDecl, err := p.consumeToken(DefaultToken)
			if err != nil {
				return nil, err
			}
			newAttribute.Add(dDecl)
			vDecl, err := p.consumeToken(FalseToken, StringToken, NumberToken, LocalTimestampToken)
			if err != nil {
				return nil, err
			}
			dDecl.Add(vDecl)
		}

		// Closing bracket ?
		if tokens[p.index].Token == BracketClosingToken {
			p.index++
			break
		}

		// Then comma ?
		if tokens[p.index].Token != CommaToken {
			return nil, p.syntaxError()
		}
		p.index++
	}

	return tableDecl, nil
}

func (p *parser) parsePrimaryKey() (*Decl, error) {
	primaryDecl, err := p.consumeToken(PrimaryToken)
	if err != nil {
		return nil, err
	}

	keyDecl, err := p.consumeToken(KeyToken)
	if err != nil {
		return nil, err
	}
	primaryDecl.Add(keyDecl)

	_, err = p.consumeToken(BracketOpeningToken)
	if err != nil {
		return nil, err
	}

	for {
		d, err := p.parseQuotedToken()
		if err != nil {
			return nil, err
		}

		d, err = p.consumeToken(CommaToken, BracketClosingToken)
		if err != nil {
			return nil, err
		}
		if d.Token == BracketClosingToken {
			break
		}
	}

	return primaryDecl, nil
}
