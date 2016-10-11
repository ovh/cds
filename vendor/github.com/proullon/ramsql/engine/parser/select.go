package parser

import (
	"fmt"
)

func (p *parser) parseSelect(tokens []Token) (*Instruction, error) {
	i := &Instruction{}
	var err error

	// Create select decl
	selectDecl := NewDecl(tokens[p.index])
	i.Decls = append(i.Decls, selectDecl)

	// After select token, should be either
	// a StarToken
	// a list of table names + (StarToken Or Attribute)
	// a builtin func (COUNT, MAX, ...)
	if err = p.next(); err != nil {
		return nil, fmt.Errorf("SELECT token must be followed by attributes to select")
	}

	for {
		if p.is(CountToken) {
			attrDecl, err := p.parseBuiltinFunc()
			if err != nil {
				return nil, err
			}
			selectDecl.Add(attrDecl)
		} else {
			attrDecl, err := p.parseAttribute()
			if err != nil {
				return nil, err
			}
			selectDecl.Add(attrDecl)
		}

		// If comma, loop again.
		if p.is(CommaToken) {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		break
	}

	// Must be from now
	if tokens[p.index].Token != FromToken {
		return nil, fmt.Errorf("Syntax error near %v\n", tokens[p.index])
	}
	fromDecl := NewDecl(tokens[p.index])
	selectDecl.Add(fromDecl)

	// Now must be a list of table
	for {
		// string
		if err = p.next(); err != nil {
			return nil, fmt.Errorf("Unexpected end. Syntax error near %v\n", tokens[p.index])
		}
		tableNameDecl, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		fromDecl.Add(tableNameDecl)

		// If no next, then it's implicit where
		if !p.hasNext() {
			addImplicitWhereAll(selectDecl)
			return i, nil
		}
		// if not comma, break
		if tokens[p.index].Token != CommaToken {
			break // No more table
		}
	}

	// JOIN OR ...?
	for p.is(JoinToken) {
		joinDecl, err := p.parseJoin()
		if err != nil {
			return nil, err
		}
		selectDecl.Add(joinDecl)
	}

	hazWhereClause := false
	for {
		switch p.cur().Token {
		case WhereToken:
			err := p.parseWhere(selectDecl)
			if err != nil {
				return nil, err
			}
			hazWhereClause = true
		case OrderToken:
			if hazWhereClause == false {
				// WHERE clause is implicit
				addImplicitWhereAll(selectDecl)
			}
			err := p.parseOrderBy(selectDecl)
			if err != nil {
				return nil, err
			}
		case LimitToken:
			limitDecl, err := p.consumeToken(LimitToken)
			if err != nil {
				return nil, err
			}
			selectDecl.Add(limitDecl)
			numDecl, err := p.consumeToken(NumberToken)
			if err != nil {
				return nil, err
			}
			limitDecl.Add(numDecl)
		case ForToken:
			err := p.parseForUpdate(selectDecl)
			if err != nil {
				return nil, err
			}
		default:
			return i, nil
		}
	}
}

func addImplicitWhereAll(decl *Decl) {

	whereDecl := &Decl{
		Token:  WhereToken,
		Lexeme: "where",
	}
	whereDecl.Add(&Decl{
		Token:  NumberToken,
		Lexeme: "1",
	})

	decl.Add(whereDecl)
}

func (p *parser) parseForUpdate(decl *Decl) error {
	// Optionnal
	if !p.is(ForToken) {
		return nil
	}

	d, err := p.consumeToken(ForToken)
	if err != nil {
		return err
	}

	u, err := p.consumeToken(UpdateToken)
	if err != nil {
		return err
	}

	d.Add(u)
	decl.Add(d)
	return nil
}
