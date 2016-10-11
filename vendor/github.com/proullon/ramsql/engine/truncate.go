package engine

import (
	"fmt"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
	"github.com/proullon/ramsql/engine/protocol"
)

func truncateExecutor(e *Engine, trDecl *parser.Decl, conn protocol.EngineConn) error {
	log.Debug("truncateExecutor")

	// get tables to be deleted
	table := NewTable(trDecl.Decl[0].Lexeme)

	return truncateTable(e, table, conn)
}

func truncateTable(e *Engine, table *Table, conn protocol.EngineConn) error {
	var rowsDeleted int64

	// get relations and write lock them
	r := e.relation(table.name)
	if r == nil {
		return fmt.Errorf("Table %v not found", table.name)
	}
	r.Lock()
	defer r.Unlock()

	if r.rows != nil {
		rowsDeleted = int64(len(r.rows))
	}
	r.rows = make([]*Tuple, 0)

	return conn.WriteResult(0, rowsDeleted)
}
