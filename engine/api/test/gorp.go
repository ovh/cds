package test

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"
)

type SqlExecutorMockQuery struct {
	Query string
	Args  []interface{}
}

type SqlExecutorMock struct {
	Queries  []SqlExecutorMockQuery
	OnSelect func(i interface{})
}

func (s SqlExecutorMock) LastQuery() SqlExecutorMockQuery {
	if len(s.Queries) == 0 {
		return SqlExecutorMockQuery{}
	}
	return s.Queries[len(s.Queries)-1]
}

func (s *SqlExecutorMock) WithContext(ctx context.Context) gorp.SqlExecutor { return s }
func (s *SqlExecutorMock) Get(i interface{}, keys ...interface{}) (interface{}, error) {
	return nil, nil
}
func (s *SqlExecutorMock) Insert(list ...interface{}) error                           { return nil }
func (s *SqlExecutorMock) Update(list ...interface{}) (int64, error)                  { return 0, nil }
func (s *SqlExecutorMock) Delete(list ...interface{}) (int64, error)                  { return 0, nil }
func (s *SqlExecutorMock) Exec(query string, args ...interface{}) (sql.Result, error) { return nil, nil }
func (s *SqlExecutorMock) Select(i interface{}, query string, args ...interface{}) ([]interface{}, error) {
	s.Queries = append(s.Queries, SqlExecutorMockQuery{
		Query: query,
		Args:  args,
	})
	if s.OnSelect != nil {
		s.OnSelect(i)
	}
	return nil, nil
}
func (s *SqlExecutorMock) SelectInt(query string, args ...interface{}) (int64, error) { return 0, nil }
func (s *SqlExecutorMock) SelectNullInt(query string, args ...interface{}) (sql.NullInt64, error) {
	return sql.NullInt64{}, nil
}
func (s *SqlExecutorMock) SelectFloat(query string, args ...interface{}) (float64, error) {
	return 0, nil
}
func (s *SqlExecutorMock) SelectNullFloat(query string, args ...interface{}) (sql.NullFloat64, error) {
	return sql.NullFloat64{}, nil
}
func (s *SqlExecutorMock) SelectStr(query string, args ...interface{}) (string, error) {
	return "", nil
}
func (s *SqlExecutorMock) SelectNullStr(query string, args ...interface{}) (sql.NullString, error) {
	return sql.NullString{}, nil
}
func (s *SqlExecutorMock) SelectOne(holder interface{}, query string, args ...interface{}) error {
	s.Queries = append(s.Queries, SqlExecutorMockQuery{
		Query: query,
		Args:  args,
	})
	return nil
}
func (s *SqlExecutorMock) Query(query string, args ...interface{}) (*sql.Rows, error) {
	s.Queries = append(s.Queries, SqlExecutorMockQuery{
		Query: query,
		Args:  args,
	})
	return nil, nil
}
func (s *SqlExecutorMock) QueryRow(query string, args ...interface{}) *sql.Row { return nil }
