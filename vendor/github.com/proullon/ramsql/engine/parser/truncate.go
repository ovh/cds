package parser

func (p *parser) parseTruncate() (*Instruction, error) {
	i := &Instruction{}

	// Set TRUNCATE decl
	trDecl, err := p.consumeToken(TruncateToken)
	if err != nil {
		return nil, err
	}
	i.Decls = append(i.Decls, trDecl)

	// Should be a table name
	nameDecl, err := p.parseQuotedToken()
	if err != nil {
		return nil, err
	}
	trDecl.Add(nameDecl)

	return i, nil
}
