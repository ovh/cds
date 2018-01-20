package zesty

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/go-gorp/gorp"
)

// Registered databases
var (
	dbs    = make(map[string]DB)
	dblock sync.RWMutex
)

type SavePoint uint

/*
 * INTERFACES
 */

type DB interface {
	gorp.SqlExecutor
	Begin() (Tx, error)
	Close() error
	Ping() error
	Stats() sql.DBStats
}

type Tx interface {
	gorp.SqlExecutor
	Commit() error
	Rollback() error
	Savepoint(string) error
	RollbackToSavepoint(string) error
}

type DBProvider interface {
	DB() gorp.SqlExecutor
	Tx() error
	TxSavepoint() (SavePoint, error)
	Commit() error
	Rollback() error
	RollbackTo(SavePoint) error
	Close() error
	Ping() error
	Stats() sql.DBStats
}

/*
 * FUNCTIONS
 */

func NewDB(dbmap *gorp.DbMap) DB {
	return &zestydb{DbMap: dbmap}
}

func RegisterDB(db DB, name string) error {
	dblock.Lock()
	defer dblock.Unlock()

	_, ok := dbs[name]
	if ok {
		return fmt.Errorf("DB name conflict '%s'", name)
	}

	dbs[name] = db

	return nil
}

func UnregisterDB(name string) error {
	dblock.Lock()
	defer dblock.Unlock()

	_, ok := dbs[name]
	if !ok {
		return fmt.Errorf("No such database '%s'", name)
	}

	delete(dbs, name)

	return nil
}

func NewDBProvider(name string) (DBProvider, error) {
	dblock.RLock()
	defer dblock.RUnlock()
	db, ok := dbs[name]
	if !ok {
		return nil, fmt.Errorf("No such database '%s'", name)
	}
	return &zestyprovider{
		current: db,
		db:      db,
	}, nil
}

func NewTempDBProvider(db DB) DBProvider {
	return &zestyprovider{
		current: db,
		db:      db,
	}
}

/*
 * PROVIDER IMPLEMENTATION
 */

type zestyprovider struct {
	current   gorp.SqlExecutor
	db        DB
	tx        Tx
	savepoint SavePoint
}

func (zp *zestyprovider) DB() gorp.SqlExecutor {
	return zp.current
}

func (zp *zestyprovider) Commit() error {
	if zp.tx == nil {
		return errors.New("No active Tx")
	}

	if zp.savepoint > 0 {
		zp.savepoint--
		return nil
	}

	err := zp.tx.Commit()
	if err != nil {
		return err
	}

	zp.resetTx()

	return nil
}

func (zp *zestyprovider) Tx() error {
	_, err := zp.TxSavepoint()
	return err
}

func (zp *zestyprovider) Rollback() error {
	return zp.RollbackTo(zp.savepoint)
}

const savepointFmt = "tx-savepoint-%d"

func (zp *zestyprovider) TxSavepoint() (SavePoint, error) {
	if zp.tx == nil {
		// root transaction
		tx, err := zp.db.Begin()
		if err != nil {
			return 0, err
		}

		zp.tx = tx
		zp.current = tx
	} else {
		// nested transaction
		s := fmt.Sprintf(savepointFmt, zp.savepoint+1)
		err := zp.tx.Savepoint(s)
		if err != nil {
			return 0, err
		}

		zp.savepoint++
	}

	return zp.savepoint, nil
}

func (zp *zestyprovider) RollbackTo(sp SavePoint) error {
	if zp.tx == nil {
		return errors.New("No active Tx")
	}
	if sp > zp.savepoint {
		// noop
		return nil
	}

	if sp == 0 {
		// root transaction
		err := zp.tx.Rollback()
		if err != nil {
			return err
		}

		zp.resetTx()
	} else {
		// nested transaction
		s := fmt.Sprintf(savepointFmt, sp)
		err := zp.tx.RollbackToSavepoint(s)
		if err != nil {
			return err
		}

		zp.savepoint = sp - 1
	}

	return nil
}

func (zp *zestyprovider) resetTx() {
	zp.current = zp.db
	zp.tx = nil
	zp.savepoint = 0
}

func (zp *zestyprovider) Close() error {
	return zp.db.Close()
}

func (zp *zestyprovider) Ping() error {
	return zp.db.Ping()
}

func (zp *zestyprovider) Stats() sql.DBStats {
	return zp.db.Stats()
}

/*
 * DATABASE IMPLEMENTATION
 */

type zestydb struct {
	*gorp.DbMap
}

func (zd *zestydb) Begin() (Tx, error) {
	return zd.DbMap.Begin()
}

func (zd *zestydb) Close() error {
	return zd.DbMap.Db.Close()
}

func (zd *zestydb) Ping() error {
	return zd.DbMap.Db.Ping()
}

func (zd *zestydb) Stats() sql.DBStats {
	return zd.DbMap.Db.Stats()
}
