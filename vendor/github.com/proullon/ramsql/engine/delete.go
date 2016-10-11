package engine

import (
	// "errors"
	"fmt"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
	"github.com/proullon/ramsql/engine/protocol"
)

func deleteExecutor(e *Engine, deleteDecl *parser.Decl, conn protocol.EngineConn) error {
	log.Debug("deleteExecutor")

	// get tables to be deleted
	tables := fromExecutor(deleteDecl.Decl[0])

	// If len is 1, it means no predicates so truncate table
	if len(deleteDecl.Decl) == 1 {
		return truncateTable(e, tables[0], conn)
	}

	// get WHERE declaration
	predicates, err := whereExecutor(deleteDecl.Decl[1], tables[0].name)
	if err != nil {
		return err
	}

	// and delete
	return deleteRows(e, tables, conn, predicates)
}

func deleteRows(e *Engine, tables []*Table, conn protocol.EngineConn, predicates []Predicate) error {
	var rowsDeleted int64

	r := e.relation(tables[0].name)
	if r == nil {
		return fmt.Errorf("Table %s not found", tables[0].name)
	}
	r.Lock()
	defer r.Unlock()

	var ok, res bool
	var err error
	lenRows := len(r.rows)
	for i := 0; i < lenRows; i++ {
		ok = true
		// If the row validate all predicates, write it
		for _, predicate := range predicates {
			if res, err = predicate.Evaluate(r.rows[i], r.table); err != nil {
				return err
			}
			if res == false {
				ok = false
				continue
			}
		}

		if ok {
			switch i {
			case 0:
				r.rows = r.rows[1:]
			case lenRows - 1:
				r.rows = r.rows[:lenRows-1]
			default:
				r.rows = append(r.rows[:i], r.rows[i+1:]...)
				i--
			}
			lenRows--
			rowsDeleted++
		}
	}

	return conn.WriteResult(0, rowsDeleted)
}
