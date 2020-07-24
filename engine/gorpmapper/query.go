package gorpmapper

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	// ViolateForeignKeyPGCode is the pg code when violating foreign key
	ViolateForeignKeyPGCode = "23503"

	// ViolateUniqueKeyPGCode is the pg code when duplicating unique key
	ViolateUniqueKeyPGCode = "23505"

	// StringDataRightTruncation is raisedalue is too long for varchar.
	StringDataRightTruncation = "22001"
)

// NewQuery returns a new query from given string request.
func NewQuery(q string) Query { return Query{Query: q} }

// Query to get gorp entities in database.
type Query struct {
	Query     string
	Arguments []interface{}
}

// Args store query arguments.
func (q Query) Args(as ...interface{}) Query {
	q.Arguments = as
	return q
}

func (q Query) Limit(i int) Query {
	q.Query += ` LIMIT ` + strconv.Itoa(i)
	return q
}

func (q Query) String() string {
	return fmt.Sprintf("query: %s - args: %v", q.Query, q.Arguments)
}

// ToQueryString returns a comma separated list of given ids.
func ToQueryString(target interface{}) string {
	val := reflect.ValueOf(target)
	if reflect.ValueOf(target).Kind() == reflect.Ptr {
		val = val.Elem()
	}

	res := make([]string, val.Len())
	for i := 0; i < val.Len(); i++ {
		res[i] = fmt.Sprintf("%v", val.Index(i).Interface())
	}

	return strings.Join(res, ",")
}
