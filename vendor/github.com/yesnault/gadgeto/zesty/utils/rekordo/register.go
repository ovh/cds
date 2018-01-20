package rekordo

import "sync"

// modelsMu protect models map.
var modelsMu sync.Mutex

// models represents the models registered
// for all databases.
var models map[string]map[string]*TableModel

func init() {
	// Initialize tables map.
	models = make(map[string]map[string]*TableModel)
}

// TableModel is a middleman between a database
// table and a model type.
type TableModel struct {
	Name          string
	Model         interface{}
	Keys          []string
	AutoIncrement bool
}

// RegisterTableModel registers a zero-value model to
// the definition of a database table. If a table model
// has already been registered with the same table name,
// this will overwrite it.
func RegisterTableModel(dbName, tableName string, model interface{}) *TableModel {
	modelsMu.Lock()
	defer modelsMu.Unlock()

	if _, ok := models[dbName]; !ok {
		// Detabase entry does not exists, let's
		// create it and add a new model for the table.
		models[dbName] = make(map[string]*TableModel)
	}
	m := &TableModel{
		Name:          tableName,
		Model:         model,
		Keys:          []string{"id"},
		AutoIncrement: true,
	}
	models[dbName][tableName] = m

	return m
}

// WithKeys uses keys as table keys for the model.
func (tb *TableModel) WithKeys(keys []string) *TableModel {
	tb.Keys = keys
	return tb
}

// WithAutoIncrement uses enable for table model keys auto-increment.
func (tb *TableModel) WithAutoIncrement(enable bool) *TableModel {
	tb.AutoIncrement = enable
	return tb
}
