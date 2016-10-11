// Package parser implements a parser for SQL statements
//
// Inspired by go/parser
package parser

import (
	"fmt"

	"github.com/proullon/ramsql/engine/log"
)

// The parser structure holds the parser's internal state.
type parser struct {
	i        []Instruction
	index    int
	tokenLen int
	tokens   []Token
}

// Decl structure is the node to statement declaration tree
type Decl struct {
	Token  int
	Lexeme string
	Decl   []*Decl
}

// Stringy prints the declaration tree in console
func (d Decl) Stringy(depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent = fmt.Sprintf("%s    ", indent)
	}

	log.Debug("%s|-> %s\n", indent, d.Lexeme)
	for _, subD := range d.Decl {
		subD.Stringy(depth + 1)
	}
}

// Instruction define a valid SQL statement
type Instruction struct {
	Decls []*Decl
}

// PrettyPrint prints instruction's declarations on console with indentation
func (i Instruction) PrettyPrint() {
	for _, d := range i.Decls {
		d.Stringy(0)
	}
}

// NewDecl initialize a Decl struct from a given token
func NewDecl(t Token) *Decl {
	return &Decl{
		Token:  t.Token,
		Lexeme: t.Lexeme,
	}
}

// Add creates a new leaf with given Decl
func (d *Decl) Add(subDecl *Decl) {
	d.Decl = append(d.Decl, subDecl)
}

func (p *parser) parse(tokens []Token) ([]Instruction, error) {
	tokens = stripSpaces(tokens)
	p.tokens = tokens
	log.Debug("parser.parse: %v\n", p.tokens)

	p.tokenLen = len(tokens)
	p.index = 0
	for p.hasNext() {
		// fmt.Printf("Token index : %d\n", p.index)

		// Found a new instruction
		if tokens[p.index].Token == SemicolonToken {
			p.index++
			continue
		}

		// Ignore space token, not needed anymore
		if tokens[p.index].Token == SpaceToken {
			p.index++
			continue
		}

		// Now,
		// Create a logical tree of all tokens
		// We start with first order query
		// CREATE, SELECT, INSERT, UPDATE, DELETE, TRUNCATE, DROP, EXPLAIN
		switch tokens[p.index].Token {
		case CreateToken:
			i, err := p.parseCreate(tokens)
			if err != nil {
				return nil, err
			}
			p.i = append(p.i, *i)
			break
		case SelectToken:
			i, err := p.parseSelect(tokens)
			if err != nil {
				return nil, err
			}
			p.i = append(p.i, *i)
			break
		case InsertToken:
			i, err := p.parseInsert()
			if err != nil {
				return nil, err
			}
			p.i = append(p.i, *i)
			break
		case UpdateToken:
			i, err := p.parseUpdate()
			if err != nil {
				return nil, err
			}
			p.i = append(p.i, *i)
			break
		case DeleteToken:
			i, err := p.parseDelete()
			if err != nil {
				return nil, err
			}
			p.i = append(p.i, *i)
			break
		case TruncateToken:
			i, err := p.parseTruncate()
			if err != nil {
				return nil, err
			}
			p.i = append(p.i, *i)
			break
		case DropToken:
			log.Debug("HEY DROP HERE !\n")
			i, err := p.parseDrop()
			if err != nil {
				return nil, err
			}
			p.i = append(p.i, *i)
			break
		case ExplainToken:
			break
		case GrantToken:
			i := &Instruction{}
			i.Decls = append(i.Decls, NewDecl(Token{Token: GrantToken}))
			p.i = append(p.i, *i)
			return p.i, nil
		default:
			return nil, fmt.Errorf("Parsing error near <%s>", tokens[p.index].Lexeme)
		}
	}

	return p.i, nil
}

func (p *parser) parseUpdate() (*Instruction, error) {
	i := &Instruction{}

	// Set DELETE decl
	updateDecl, err := p.consumeToken(UpdateToken)
	if err != nil {
		return nil, err
	}
	i.Decls = append(i.Decls, updateDecl)

	// should be table name
	nameDecl, err := p.parseQuotedToken()
	if err != nil {
		return nil, err
	}
	updateDecl.Add(nameDecl)

	// should be SET
	setDecl, err := p.consumeToken(SetToken)
	if err != nil {
		return nil, err
	}
	updateDecl.Add(setDecl)

	// should be a list of equality
	gotClause := false
	for p.tokens[p.index].Token != WhereToken {

		if !p.hasNext() && gotClause {
			break
		}

		attributeDecl, err := p.parseCondition()
		if err != nil {
			return nil, err
		}
		setDecl.Add(attributeDecl)
		p.consumeToken(CommaToken)

		// Got at least one clause
		gotClause = true
	}

	err = p.parseWhere(updateDecl)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (p *parser) parseInsert() (*Instruction, error) {
	i := &Instruction{}

	// Set INSERT decl
	insertDecl, err := p.consumeToken(InsertToken)
	if err != nil {
		return nil, err
	}
	i.Decls = append(i.Decls, insertDecl)

	// should be INTO
	intoDecl, err := p.consumeToken(IntoToken)
	if err != nil {
		return nil, err
	}
	insertDecl.Add(intoDecl)

	// should be table Name
	tableDecl, err := p.parseQuotedToken()
	if err != nil {
		return nil, err
	}
	intoDecl.Add(tableDecl)

	_, err = p.consumeToken(BracketOpeningToken)
	if err != nil {
		return nil, err
	}

	// concerned attribute
	for {
		decl, err := p.parseListElement()
		if err != nil {
			return nil, err
		}
		tableDecl.Add(decl)

		if p.is(BracketClosingToken) {
			if _, err = p.consumeToken(BracketClosingToken); err != nil {
				return nil, err
			}

			break
		}

		_, err = p.consumeToken(CommaToken)
		if err != nil {
			return nil, err
		}
	}

	// should be VALUES
	valuesDecl, err := p.consumeToken(ValuesToken)
	if err != nil {
		return nil, err
	}
	insertDecl.Add(valuesDecl)

	_, err = p.consumeToken(BracketOpeningToken)
	if err != nil {
		return nil, err
	}

	// should be a list of values for specified attributes
	for {
		decl, err := p.parseListElement()
		if err != nil {
			return nil, err
		}
		valuesDecl.Add(decl)

		if p.is(BracketClosingToken) {
			p.consumeToken(BracketClosingToken)
			break
		}

		_, err = p.consumeToken(CommaToken)
		if err != nil {
			return nil, err
		}
	}

	// we may have `returning "something"` here
	if retDecl, err := p.consumeToken(ReturningToken); err == nil {
		insertDecl.Add(retDecl)
		attrDecl, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		retDecl.Add(attrDecl)
	}

	return i, nil
}

func (p *parser) parseType() (*Decl, error) {
	typeDecl, err := p.consumeToken(StringToken)
	if err != nil {
		return nil, err
	}

	// Maybe a complex type
	if p.is(BracketOpeningToken) {
		_, err = p.consumeToken(BracketOpeningToken)
		if err != nil {
			return nil, err
		}
		sizeDecl, err := p.consumeToken(NumberToken)
		if err != nil {
			return nil, err
		}
		typeDecl.Add(sizeDecl)
		_, err = p.consumeToken(BracketClosingToken)
		if err != nil {
			return nil, err
		}
	}

	return typeDecl, nil
}

func (p *parser) parseOrderBy(selectDecl *Decl) error {
	orderDecl, err := p.consumeToken(OrderToken)
	if err != nil {
		return err
	}
	selectDecl.Add(orderDecl)

	_, err = p.consumeToken(ByToken)
	if err != nil {
		return err
	}

	// parse attribute now
	attrDecl, err := p.parseAttribute()
	if err != nil {
		return err
	}
	orderDecl.Add(attrDecl)

	// Parse multiple ordering
	for p.cur().Token == CommaToken {
		_, err := p.consumeToken(CommaToken)
		if err != nil {
			return nil
		}

		// parse attribute now
		attrDecl, err := p.parseAttribute()
		if err != nil {
			return err
		}
		orderDecl.Add(attrDecl)
	}

	// ASC ? DESC ? nothing ?
	t := p.cur().Token
	if t == AscToken || t == DescToken {
		decl, err := p.consumeToken(AscToken, DescToken)
		if err != nil {
			return err
		}
		orderDecl.Add(decl)
	}

	return nil
}

func (p *parser) parseWhere(selectDecl *Decl) error {

	// May be WHERE  here
	// Can be ORDER BY if WHERE cause if implicit
	whereDecl, err := p.consumeToken(WhereToken)
	if err != nil {
		return err
	}
	selectDecl.Add(whereDecl)

	// Now should be a list of: Attribute and Operator and Value
	gotClause := false
	for {
		if !p.hasNext() && gotClause {
			break
		}

		if p.is(OrderToken, LimitToken, ForToken) {
			break
		}

		attributeDecl, err := p.parseCondition()
		if err != nil {
			return err
		}
		whereDecl.Add(attributeDecl)

		if p.is(AndToken, OrToken) {
			linkDecl, err := p.consumeToken(p.cur().Token)
			if err != nil {
				return err
			}
			whereDecl.Add(linkDecl)
		}

		// Got at least one clause
		gotClause = true
	}

	return nil
}

// parseBuiltinFunc looks for COUNT,MAX,MIN
func (p *parser) parseBuiltinFunc() (*Decl, error) {
	var d *Decl
	var err error

	// COUNT(attribute)
	if p.is(CountToken) {
		d, err = p.consumeToken(CountToken)
		if err != nil {
			return nil, err
		}
		// Bracket
		_, err = p.consumeToken(BracketOpeningToken)
		if err != nil {
			return nil, err
		}
		// Attribute
		attr, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		d.Add(attr)
		// Bracket
		_, err = p.consumeToken(BracketClosingToken)
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

// parseAttribute parse an attribute of the form
// table.foo
// table.*
// "table".foo
// foo
func (p *parser) parseAttribute() (*Decl, error) {
	quoted := false

	if p.is(DoubleQuoteToken) {
		quoted = true
		if err := p.next(); err != nil {
			return nil, err
		}
	}

	// shoud be a StringToken here
	// If there is a point after, it's a table name,
	// if not, it's the attribute
	if !p.is(StringToken, StarToken) {
		return nil, p.syntaxError()
	}
	decl := NewDecl(p.cur())

	if quoted {
		// Check there is a closing quote
		if _, err := p.mustHaveNext(DoubleQuoteToken); err != nil {
			log.Debug("parseAttribute: Missing closing quote")
			return nil, err
		}
	}
	// If no next token,and not quoted, then is was the atribute name
	if err := p.next(); err != nil {
		return decl, nil
	}

	// Now, is it a point ?
	if p.is(PeriodToken) {
		_, err := p.consumeToken(PeriodToken)
		if err != nil {
			return nil, err
		}
		// if so, next must be the attribute name or a star
		attributeDecl, err := p.consumeToken(StringToken, StarToken)
		if err != nil {
			return nil, err
		}
		attributeDecl.Add(decl)
		return attributeDecl, nil
	}

	// Then the first string token was the naked attribute name
	return decl, nil
}

// parseQuotedToken parse a token of the form
// table
// "table"
func (p *parser) parseQuotedToken() (*Decl, error) {
	quoted := false

	if p.is(DoubleQuoteToken) {
		quoted = true
		if err := p.next(); err != nil {
			return nil, err
		}
	}

	// shoud be a StringToken here
	if !p.is(StringToken) {
		return nil, p.syntaxError()
	}
	decl := NewDecl(p.cur())

	if quoted {

		// Check there is a closing quote
		if _, err := p.mustHaveNext(DoubleQuoteToken); err != nil {
			return nil, err
		}
	}

	p.next()
	return decl, nil
}

func (p *parser) parseCondition() (*Decl, error) {

	// We may have the WHERE 1 condition
	if t := p.cur(); t.Token == NumberToken && t.Lexeme == "1" {
		attributeDecl := NewDecl(t)
		p.next()
		// in case of 1 = 1
		if p.cur().Token == EqualityToken {
			t, err := p.isNext(NumberToken)
			if err == nil && t.Lexeme == "1" {
				p.consumeToken(EqualityToken)
				p.consumeToken(NumberToken)
			}
		}
		return attributeDecl, nil
	}

	// Attribute
	attributeDecl, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}

	switch p.cur().Token {
	case EqualityToken, LeftDipleToken, RightDipleToken:
		decl, err := p.consumeToken(p.cur().Token)
		if err != nil {
			return nil, err
		}
		attributeDecl.Add(decl)
		break
	case InToken:
		inDecl, err := p.parseIn()
		if err != nil {
			return nil, err
		}
		attributeDecl.Add(inDecl)
		return attributeDecl, nil
	case IsToken:
		log.Debug("parseCondition: IsToken\n")
		decl, err := p.consumeToken(IsToken)
		if err != nil {
			return nil, err
		}
		attributeDecl.Add(decl)
		if p.cur().Token == NotToken {
			log.Debug("parseCondition: NotToken\n")
			notDecl, err := p.consumeToken(NotToken)
			if err != nil {
				return nil, err
			}
			decl.Add(notDecl)
		}
		if p.cur().Token == NullToken {
			log.Debug("parseCondition: NullToken\n")
			nullDecl, err := p.consumeToken(NullToken)
			if err != nil {
				return nil, err
			}
			decl.Add(nullDecl)
		}
		return attributeDecl, nil
	}

	// Value
	valueDecl, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	attributeDecl.Add(valueDecl)
	return attributeDecl, nil
}

func (p *parser) parseIn() (*Decl, error) {
	inDecl, err := p.consumeToken(InToken)
	if err != nil {
		return nil, err
	}

	// bracket opening
	_, err = p.consumeToken(BracketOpeningToken)
	if err != nil {
		return nil, err
	}

	// list of value
	gotList := false
	for {
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		inDecl.Add(v)
		gotList = true

		if p.is(BracketClosingToken) {
			if gotList == false {
				return nil, fmt.Errorf("IN clause: empty list of value")
			}
			p.consumeToken(BracketClosingToken)
			break
		}

		_, err = p.consumeToken(CommaToken)
		if err != nil {
			return nil, err
		}
	}

	return inDecl, nil
}

func (p *parser) parseValue() (*Decl, error) {
	debug("parseValue")
	defer debug("~parseValue")
	quoted := false

	if p.is(SimpleQuoteToken) || p.is(DoubleQuoteToken) {
		quoted = true
		debug("value %v is quoted!", p.tokens[p.index])
		_, err := p.consumeToken(SimpleQuoteToken, DoubleQuoteToken)
		if err != nil {
			return nil, err
		}
	}

	valueDecl, err := p.consumeToken(StringToken, NumberToken, DateToken, NowToken)
	if err != nil {
		debug("parseValue: Wasn't expecting %v\n", p.tokens[p.index])
		return nil, err
	}
	log.Debug("Parsing value %v !\n", valueDecl)

	if quoted {
		log.Debug("consume quote %v\n", p.tokens[p.index])
		_, err := p.consumeToken(SimpleQuoteToken, DoubleQuoteToken)
		if err != nil {
			debug("uuuh, wasn't a quote")
			return nil, err
		}
	}

	return valueDecl, nil
}

// parseJoin parses the JOIN keywords and all its condition
// JOIN user_addresses ON address.id=user_addresses.address_id
func (p *parser) parseJoin() (*Decl, error) {
	joinDecl, err := p.consumeToken(JoinToken)
	if err != nil {
		return nil, err
	}

	// TABLE NAME
	tableDecl, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}
	joinDecl.Add(tableDecl)

	// ON
	onDecl, err := p.consumeToken(OnToken)
	if err != nil {
		return nil, err
	}
	// onDecl := NewDecl(t)
	joinDecl.Add(onDecl)

	// ATTRIBUTE
	leftAttributeDecl, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}
	onDecl.Add(leftAttributeDecl)

	// EQUAL
	equalAttr, err := p.consumeToken(EqualityToken)
	if err != nil {
		return nil, err
	}
	onDecl.Add(equalAttr)

	//ATTRIBUTE
	rightAttributeDecl, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}
	onDecl.Add(rightAttributeDecl)

	return joinDecl, nil
}

func (p *parser) parseListElement() (*Decl, error) {
	quoted := false

	// In case of INSERT, can be DEFAULT here
	if p.is(DefaultToken) {
		v, err := p.consumeToken(DefaultToken)
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	if p.is(SimpleQuoteToken) || p.is(DoubleQuoteToken) {
		quoted = true
		p.next()
	}

	var valueDecl *Decl
	valueDecl, err := p.consumeToken(StringToken, NumberToken, NullToken, DateToken, NowToken)
	if err != nil {
		return nil, err
	}

	if quoted {
		if _, err := p.consumeToken(SimpleQuoteToken, DoubleQuoteToken); err != nil {
			return nil, err
		}
	}

	return valueDecl, nil
}

func (p *parser) next() error {
	if !p.hasNext() {
		return fmt.Errorf("Unexpected end")
	}
	p.index++
	return nil
}

func (p *parser) hasNext() bool {
	if p.index+1 < len(p.tokens) {
		return true
	}
	return false
}

func (p *parser) is(tokenTypes ...int) bool {

	for _, tokenType := range tokenTypes {
		if p.cur().Token == tokenType {
			return true
		}
	}

	return false
}

func (p *parser) isNext(tokenTypes ...int) (t Token, err error) {

	if !p.hasNext() {
		debug("parser.isNext: has no next")
		return t, p.syntaxError()
	}

	debug("parser.isNext %v", tokenTypes)
	for _, tokenType := range tokenTypes {
		if p.tokens[p.index+1].Token == tokenType {
			return p.tokens[p.index+1], nil
		}
	}

	debug("parser.isNext: Next (%v) is not among %v", p.cur(), tokenTypes)
	return t, p.syntaxError()
}

func (p *parser) mustHaveNext(tokenTypes ...int) (t Token, err error) {

	if !p.hasNext() {
		debug("parser.mustHaveNext: has no next")
		return t, p.syntaxError()
	}

	if err = p.next(); err != nil {
		debug("parser.mustHaveNext: error getting next")
		return t, err
	}

	debug("parser.mustHaveNext %v", tokenTypes)
	for _, tokenType := range tokenTypes {
		if p.is(tokenType) {
			return p.tokens[p.index], nil
		}
	}

	debug("parser.mustHaveNext: Next (%v) is not among %v", p.cur(), tokenTypes)
	return t, p.syntaxError()
}

func (p *parser) cur() Token {
	return p.tokens[p.index]
}

func (p *parser) consumeToken(tokenTypes ...int) (*Decl, error) {

	if !p.is(tokenTypes...) {
		return nil, p.syntaxError()
	}

	decl := NewDecl(p.tokens[p.index])
	p.next()
	return decl, nil
}

func (p *parser) syntaxError() error {
	if p.index == 0 {
		return fmt.Errorf("Syntax error near %v %v", p.tokens[p.index].Lexeme, p.tokens[p.index+1].Lexeme)
	} else if !p.hasNext() {
		return fmt.Errorf("Syntax error near %v %v", p.tokens[p.index-1].Lexeme, p.tokens[p.index].Lexeme)
	}
	return fmt.Errorf("Syntax error near %v %v %v", p.tokens[p.index-1].Lexeme, p.tokens[p.index].Lexeme, p.tokens[p.index+1].Lexeme)
}

func stripSpaces(t []Token) (ret []Token) {
	for i := range t {
		if t[i].Token != SpaceToken {
			ret = append(ret, t[i])
		}
	}
	return ret
}
