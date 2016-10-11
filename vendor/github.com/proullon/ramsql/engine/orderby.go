package engine

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
	"github.com/proullon/ramsql/engine/protocol"
)

//    |-> order
//        |-> age
//        |-> desc
func orderbyExecutor(attr *parser.Decl, tables []*Table) (selectFunctor, error) {
	f := &orderbyFunctor{}
	f.buffer = make(map[int64][][]string)

	// first subdecl should be attribute
	if len(attr.Decl) < 1 {
		return nil, fmt.Errorf("ordering attribute not provided")
	}

	// FIXME we should find for sure the table of the attribute
	if len(tables) < 1 {
		return nil, fmt.Errorf("cannot guess the table of attribute %s for order", attr.Decl[0].Lexeme)
	}
	if len(attr.Decl[0].Decl) > 0 {
		f.orderby = attr.Decl[0].Decl[0].Lexeme + "." + attr.Decl[0].Lexeme
	} else {
		f.orderby = tables[0].name + "." + attr.Decl[0].Lexeme
	}
	// if second subdecl is present, it's either asc or desc
	// default is asc anyway
	if len(attr.Decl) == 2 && attr.Decl[1].Token == parser.AscToken {
		f.asc = true
	}

	log.Debug("orderbyExecutor> you must order by '%s', asc: %v\n", f.orderby, f.asc)
	return f, nil
}

// ok so our buffer is a map of INDEX -> slice of ROW
// let's say we can only order by integer values
// and yeah we can have multiple row with one value, order is then random
type orderbyFunctor struct {
	e          *Engine
	conn       protocol.EngineConn
	attributes []string
	alias      []string
	orderby    string
	asc        bool
	buffer     map[int64][][]string
	order      orderer
}

func (f *orderbyFunctor) Init(e *Engine, conn protocol.EngineConn, attr []string, alias []string) error {
	f.e = e
	f.conn = conn
	f.attributes = attr
	f.alias = alias

	return f.conn.WriteRowHeader(f.alias)
}

func (f *orderbyFunctor) FeedVirtualRow(vrow virtualRow) error {

	// search key
	val, ok := vrow[f.orderby]
	if !ok {
		return fmt.Errorf("could not find ordering attribute %s in virtual row", f.orderby)
	}

	if f.order == nil { // first time
		o, err := initOrderer(val, f.attributes)
		if err != nil {
			return err
		}
		f.order = o
	}

	return f.order.Feed(val, vrow)
}

func (f *orderbyFunctor) Done() error {
	log.Debug("orderByFunctor.Done\n")

	// No row in result set, orderer hasn't been initialized
	if f.order == nil {
		return f.conn.WriteRowEnd()
	}

	if f.asc {
		f.order.Sort()
	} else {
		f.order.SortReverse()
	}

	err := f.order.Write(f.conn)
	if err != nil {
		return err
	}

	return f.conn.WriteRowEnd()
}

type sortSlice []int64

func (s sortSlice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s sortSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortSlice) Len() int {
	return len(s)
}

type orderer interface {
	Feed(key Value, vrow virtualRow) error
	Sort() error
	SortReverse() error
	Write(conn protocol.EngineConn) error
}

func initOrderer(val Value, attr []string) (orderer, error) {
	log.Debug("initOrder: %v\n", val)
	_, err := strconv.ParseInt(fmt.Sprintf("%v", val.v), 10, 64)
	if err == nil {
		log.Debug("initOrderer> key is in fact an integer\n")
		i := &intOrderer{}
		i.init(attr)
		return i, nil
	}

	/* OK SO
	 * Is the key an integer, a string or a date ?
	 */
	switch v := val.v.(type) {
	case string:
		s := &stringOrderer{}
		s.init(attr)
		return s, nil
	case int, int64:
		i := &intOrderer{}
		i.init(attr)
		return i, nil
	/*case time.Time:
	d := dateOrderer{}
	d.init()
	return d*/
	default:
		return nil, fmt.Errorf("cannot order %T with value %v", val.v, v)
	}
}

type stringOrderer struct {
	buffer     map[string][][]string
	attributes []string
	keys       []string
}

func (i *stringOrderer) init(attr []string) {
	i.buffer = make(map[string][][]string)
	i.attributes = attr
}

func (i *stringOrderer) Feed(val Value, vrow virtualRow) error {
	var row []string

	key, ok := val.v.(string)
	if !ok {
		return fmt.Errorf("error ordering because of value %v", val.v)
	}

	for _, attr := range i.attributes {
		val, ok := vrow[attr]
		if !ok {
			return fmt.Errorf("could not select attribute %s", attr)
		}
		row = append(row, fmt.Sprintf("%v", val.v))
	}

	// now instead of writing row, we will find the ordering key and put in in our buffer
	i.buffer[key] = append(i.buffer[key], row)
	return nil
}

func (i *stringOrderer) Sort() error {
	// now we have to sort our key
	i.keys = make([]string, len(i.buffer))
	var index int64
	for k := range i.buffer {
		i.keys[index] = k
		index++
	}

	sort.Sort(sort.StringSlice(i.keys))
	return nil
}

func (i *stringOrderer) SortReverse() error {
	// now we have to sort our key
	i.keys = make([]string, len(i.buffer))
	var index int64
	for k := range i.buffer {
		i.keys[index] = k
		index++
	}

	sort.Sort(sort.Reverse(sort.StringSlice(i.keys)))
	return nil
}

func (i *stringOrderer) Write(conn protocol.EngineConn) error {
	// now write ordered rows
	for _, key := range i.keys {
		rows := i.buffer[key]
		for index := range rows {
			err := conn.WriteRow(rows[index])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type intOrderer struct {
	buffer     map[int64][][]string
	attributes []string
	keys       []int64
}

func (i *intOrderer) init(attr []string) {
	i.buffer = make(map[int64][][]string)
	i.attributes = attr
}

func (i *intOrderer) Feed(val Value, vrow virtualRow) error {
	var row []string
	var key int64
	var err error

	key, err = strconv.ParseInt(fmt.Sprintf("%v", val.v), 10, 64)
	if err != nil {
		return fmt.Errorf("error ordering because of value %v", val.v)
	}

	for _, attr := range i.attributes {
		val, ok := vrow[attr]
		if !ok {
			return fmt.Errorf("could not select attribute %s", attr)
		}
		row = append(row, fmt.Sprintf("%v", val.v))
	}

	// now instead of writing row, we will find the ordering key and put in in our buffer
	i.buffer[key] = append(i.buffer[key], row)
	return nil
}

func (i *intOrderer) Sort() error {
	// now we have to sort our key
	i.keys = make([]int64, len(i.buffer))
	var index int64
	for k := range i.buffer {
		i.keys[index] = k
		index++
	}

	sort.Sort(sortSlice(i.keys))
	return nil
}

func (i *intOrderer) SortReverse() error {
	// now we have to sort our key
	i.keys = make([]int64, len(i.buffer))
	var index int64
	for k := range i.buffer {
		i.keys[index] = k
		index++
	}

	sort.Sort(sort.Reverse(sortSlice(i.keys)))
	return nil
}

func (i *intOrderer) Write(conn protocol.EngineConn) error {
	// now write ordered rows
	for _, key := range i.keys {
		rows := i.buffer[key]
		for index := range rows {
			conn.WriteRow(rows[index])
		}
	}
	return nil
}

type dateOrderer struct {
	buffer map[time.Time][][]string
}
