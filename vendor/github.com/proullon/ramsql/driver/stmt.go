package ramsql

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/proullon/ramsql/engine/log"
)

// Stmt implements the Statement interface of sql/driver
type Stmt struct {
	conn     *Conn
	query    string
	numInput int
}

func countArguments(query string) int {
	for id := 1; id > 0; id++ {
		sep := fmt.Sprintf("$%d", id)
		if strings.Count(query, sep) == 0 {
			return id - 1
		}
	}

	return -1
}

func prepareStatement(c *Conn, query string) *Stmt {

	// Parse number of arguments here
	// Should handler either Postgres ($*) or ODBC (?) parameter markers
	numInput := strings.Count(query, "?")
	// if numInput == 0, maybe it's Postgres format
	if numInput == 0 {
		numInput = countArguments(query)
	}

	// Create statement
	stmt := &Stmt{
		conn:     c,
		query:    query,
		numInput: numInput,
	}

	stmt.conn.mutex.Lock()
	return stmt
}

// Close closes the statement.
//
// As of Go 1.1, a Stmt will not be closed if it's in use
// by any queries.
func (s *Stmt) Close() error {
	return fmt.Errorf("Not implemented.")
}

// NumInput returns the number of placeholder parameters.
//
// If NumInput returns >= 0, the sql package will sanity check
// argument counts from callers and return errors to the caller
// before the statement's Exec or Query methods are called.
//
// NumInput may also return -1, if the driver doesn't know
// its number of placeholders. In that case, the sql package
// will not sanity check Exec or Query argument counts.
func (s *Stmt) NumInput() int {
	return s.numInput
}

// Exec executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
func (s *Stmt) Exec(args []driver.Value) (r driver.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fatalf error: %s", r)
			return
		}
	}()
	defer s.conn.mutex.Unlock()

	var finalQuery string

	// replace $* by arguments in query string
	finalQuery = replaceArguments(s.query, args)
	log.Info("Exec <%s>\n", finalQuery)

	// Send query to server
	err = s.conn.conn.WriteExec(finalQuery)
	if err != nil {
		log.Warning("Exec: Cannot send query to server: %s", err)
		return nil, fmt.Errorf("Cannot send query to server: %s", err)
	}

	// Get answer from server
	lastInsertedID, rowsAffected, err := s.conn.conn.ReadResult()
	if err != nil {
		return nil, err
	}

	// Create a driver.Result
	return newResult(lastInsertedID, rowsAffected), nil
}

// Query executes a query that may return rows, such as a
// SELECT.
func (s *Stmt) Query(args []driver.Value) (r driver.Rows, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fatalf error: %s", r)
			return
		}
	}()
	defer s.conn.mutex.Unlock()

	finalQuery := replaceArguments(s.query, args)
	log.Info("Query <%s>\n", finalQuery)
	err = s.conn.conn.WriteQuery(finalQuery)
	if err != nil {
		return nil, err
	}

	rowsChannel, err := s.conn.conn.ReadRows()
	if err != nil {
		return nil, err
	}

	r = newRows(rowsChannel)
	return r, nil
}

// replace $* by arguments in query string
func replaceArguments(query string, args []driver.Value) string {

	holder := regexp.MustCompile(`[^\$]\$[0-9]+`)
	replacedQuery := ""

	if strings.Count(query, "?") == len(args) {
		return replaceArgumentsODBC(query, args)
	}

	allloc := holder.FindAllIndex([]byte(query), -1)
	queryB := []byte(query)
	for i, loc := range allloc {
		match := queryB[loc[0]+1 : loc[1]]

		index, err := strconv.Atoi(string(match[1:]))
		if err != nil {
			log.Warning("Matched %s as a placeholder but cannot get index: %s\n", match, err)
			return query
		}

		var v string
		if args[index-1] == nil {
			v = "null"
		} else {
			v = fmt.Sprintf("$$%v$$", args[index-1])
		}
		if i == 0 {
			replacedQuery = fmt.Sprintf("%s%s%s", replacedQuery, string(queryB[:loc[0]+1]), v)
		} else {
			replacedQuery = fmt.Sprintf("%s%s%s", replacedQuery, string(queryB[allloc[i-1][1]:loc[0]+1]), v)
		}
	}
	// add remaining query
	replacedQuery = fmt.Sprintf("%s%s", replacedQuery, string(queryB[allloc[len(allloc)-1][1]:]))

	return replacedQuery
}

func replaceArgumentsODBC(query string, args []driver.Value) string {
	var finalQuery string

	queryParts := strings.Split(query, "?")
	finalQuery = queryParts[0]
	for i := range args {
		arg := fmt.Sprintf("%v", args[i])
		_, ok := args[i].(string)
		if ok && !strings.HasSuffix(query, "'") {
			arg = "$$" + arg + "$$"
		}
		finalQuery += arg
		finalQuery += queryParts[i+1]
	}

	return finalQuery
}
