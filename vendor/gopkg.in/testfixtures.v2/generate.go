package testfixtures

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"unicode/utf8"

	"gopkg.in/yaml.v2"
)

// GenerateFixtures generates fixtures for the current contents of a database, and saves
// them to the specified directory
func GenerateFixtures(db *sql.DB, helper Helper, dir string) error {
	tables, err := helper.tableNames(db)
	if err != nil {
		return err
	}
	for _, table := range tables {
		filename := path.Join(dir, table+".yml")
		if err := generateFixturesForTable(db, helper, table, filename); err != nil {
			return err
		}
	}
	return nil
}

func generateFixturesForTable(db *sql.DB, h Helper, table string, filename string) error {
	query := fmt.Sprintf("SELECT * FROM %s", h.quoteKeyword(table))
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	fixtures := make([]interface{}, 0, 10)
	for rows.Next() {
		entries := make([]interface{}, len(columns))
		entryPtrs := make([]interface{}, len(entries))
		for i := range entries {
			entryPtrs[i] = &entries[i]
		}
		if err := rows.Scan(entryPtrs...); err != nil {
			return err
		}

		entryMap := make(map[string]interface{}, len(entries))
		for i, column := range columns {
			entryMap[column] = convertValue(entries[i])
		}
		fixtures = append(fixtures, entryMap)
	}
	if err = rows.Err(); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	marshaled, err := yaml.Marshal(fixtures)
	if err != nil {
		return err
	}
	_, err = f.Write(marshaled)
	return err
}

func convertValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []byte:
		if utf8.Valid(v) {
			return string(v)
		}
	}
	return value
}
