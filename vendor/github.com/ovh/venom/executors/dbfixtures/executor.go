package dbfixtures

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/mitchellh/mapstructure"
	migrate "github.com/rubenv/sql-migrate"
	fixtures "gopkg.in/testfixtures.v2"
	// SQL drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name of the executor.
const Name = "dbfixtures"

// New returns a new executor that can load
// database fixtures.
func New() venom.Executor {
	return &Executor{}
}

// Executor is a venom executor that can load
// fixtures in many databases, using YAML schemas.
type Executor struct {
	Files      []string `json:"files" yaml:"files"`
	Folder     string   `json:"folder" yaml:"folder"`
	Database   string   `json:"database" yaml:"database"`
	DSN        string   `json:"dsn" yaml:"dsn"`
	Schemas    []string `json:"schemas" yaml:"schemas"`
	Migrations string   `json:"migrations" yaml:"migrations"`
}

// Result represents a step result.
type Result struct {
	Executor Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
}

// Run implements the venom.Executor interface for Executor.
func (e Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep, workdir string) (venom.ExecutorResult, error) {
	// Transform step to Executor instance.
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	// Connect to the database and ping it.
	l.Debugf("connecting to database %s, %s\n", e.Database, e.DSN)

	db, err := sql.Open(e.Database, e.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	// Load and import the schemas in the database
	// if the argument is specified.
	if len(e.Schemas) != 0 {
		for _, s := range e.Schemas {
			l.Debugf("loading schema from file %s\n", s)
			s = path.Join(workdir, s)
			sbytes, errs := ioutil.ReadFile(s)
			if errs != nil {
				return nil, errs
			}
			if _, err = db.Exec(string(sbytes)); err != nil {
				return nil, fmt.Errorf("failed to exec schema from file %s : %v", s, err)
			}
		}
	} else if e.Migrations != "" {
		l.Debugf("loading migrations from folder %s\n", e.Migrations)

		migrations := &migrate.FileMigrationSource{
			Dir: e.Migrations,
		}
		n, errMigrate := migrate.Exec(db, e.Database, migrations, migrate.Up)
		if errMigrate != nil {
			return nil, fmt.Errorf("failed to apply up migrations: %s", errMigrate)
		}
		l.Debugf("applied %d migrations\n", n)
	}
	// Load fixtures in the databases.
	// Bu default the package refuse to load if the database
	// does not contains test to avoid wiping a production db.
	fixtures.SkipDatabaseNameCheck(true)
	if err = loadFixtures(db, e.Files, e.Folder, databaseHelper(e.Database), l, workdir); err != nil {
		return nil, err
	}
	r := Result{Executor: e}

	return executors.Dump(r)
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return the default assertions of the executor.
func (e Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{}}
}

// loadFixtures loads the fixtures in the database.
// It gives priority to the fixtures files found in folder,
// and switch to the list of files if no folder was specified.
func loadFixtures(db *sql.DB, files []string, folder string, helper fixtures.Helper, l venom.Logger, workdir string) error {
	if folder != "" {
		l.Debugf("loading fixtures from folder %s\n", path.Join(workdir, folder))

		c, err := fixtures.NewFolder(db, helper, path.Join(workdir, folder))
		if err != nil {
			return fmt.Errorf("failed to create folder context: %v", err)
		}
		if err = c.Load(); err != nil {
			return fmt.Errorf("failed to load fixtures from folder %s: %v", path.Join(workdir, folder), err)
		}
		return nil
	}
	if len(files) != 0 {
		l.Debugf("loading fixtures from files: %v\n", files)
		for i := range files {
			files[i] = path.Join(workdir, files[i])
		}
		c, err := fixtures.NewFiles(db, helper, files...)
		if err != nil {
			return fmt.Errorf("failed to create files context: %v", err)
		}
		if err = c.Load(); err != nil {
			return fmt.Errorf("failed to load fixtures from files: %v", err)
		}
		return nil
	}
	l.Debugf("neither files or folder parameter was used\n")

	return nil
}

func databaseHelper(name string) fixtures.Helper {
	switch name {
	case "postgres":
		return &fixtures.PostgreSQL{}
	case "mysql":
		return &fixtures.MySQL{}
	}
	return nil
}
