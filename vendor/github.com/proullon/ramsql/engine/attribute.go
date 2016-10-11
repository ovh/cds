package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
)

// Domain is the set of allowable values for an Attribute.
type Domain struct {
}

// Attribute is a named column of a relation
// AKA Field
// AKA Column
type Attribute struct {
	name          string
	typeName      string
	typeInstance  interface{}
	defaultValue  interface{}
	domain        Domain
	autoIncrement bool
}

func parseAttribute(decl *parser.Decl) (Attribute, error) {
	attr := Attribute{}

	// Attribute name
	if decl.Token != parser.StringToken {
		return attr, fmt.Errorf("engine: expected attribute name, got %v", decl.Token)
	}
	attr.name = decl.Lexeme

	// Attribute type
	if len(decl.Decl) < 1 {
		return attr, fmt.Errorf("Attribute %s has no type", decl.Lexeme)
	}
	if decl.Decl[0].Token != parser.StringToken {
		return attr, fmt.Errorf("engine: expected attribute type, got %v:%v", decl.Decl[0].Token, decl.Decl[0].Lexeme)
	}
	attr.typeName = decl.Decl[0].Lexeme

	// Maybe domain and special thing like primary key
	typeDecl := decl.Decl[1:]
	for i := range typeDecl {
		log.Debug("Got %v for %s %s", typeDecl[i], attr.name, attr.typeName)
		if typeDecl[i].Token == parser.AutoincrementToken {
			attr.autoIncrement = true
		}

		if typeDecl[i].Token == parser.DefaultToken {
			log.Debug("we get a default value for %s: %s!\n", attr.name, typeDecl[i].Decl[0].Lexeme)
			switch typeDecl[i].Decl[0].Token {
			case parser.LocalTimestampToken, parser.NowToken:
				log.Debug("Setting default value to NOW() func !\n")
				attr.defaultValue = func() interface{} { return time.Now() }
			default:
				log.Debug("Setting default value to '%v'\n", typeDecl[i].Decl[0].Lexeme)
				attr.defaultValue = typeDecl[i].Decl[0].Lexeme
			}
		}

	}

	if strings.ToLower(attr.typeName) == "bigserial" {
		attr.autoIncrement = true
	}

	return attr, nil
}

// NewAttribute initialize a new Attribute struct
func NewAttribute(name string, typeName string, autoIncrement bool) Attribute {
	a := Attribute{
		name:          name,
		typeName:      typeName,
		autoIncrement: autoIncrement,
	}

	return a
}
