package engine

import (
	"fmt"

	"github.com/proullon/ramsql/engine/parser"
	"github.com/proullon/ramsql/engine/protocol"
)

func ifExecutor(e *Engine, ifDecl *parser.Decl, conn protocol.EngineConn) error {

	if len(ifDecl.Decl) == 0 {
		return fmt.Errorf("malformed condition")
	}

	if e.opsExecutors[ifDecl.Decl[0].Token] != nil {
		return e.opsExecutors[ifDecl.Decl[0].Token](e, ifDecl.Decl[0], conn)
	}

	return fmt.Errorf("error near %v, unknown keyword", ifDecl.Decl[0].Lexeme)
}

func notExecutor(e *Engine, tableDecl *parser.Decl, conn protocol.EngineConn) error {
	return nil
}

func existsExecutor(e *Engine, tableDecl *parser.Decl, conn protocol.EngineConn) error {
	return nil
}
