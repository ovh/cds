package services

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// FindByTokenID a service by its token_id
func FindByTokenID(db gorp.SqlExecutor, tokenID string) (*sdk.Service, error) {
	query := "SELECT * FROM services WHERE token_id = $1"
	return findOne(db, query, tokenID)
}

// FindByNameAndType a service by its name and type
func FindByNameAndType(db gorp.SqlExecutor, name, stype string) (*sdk.Service, error) {
	query := "SELECT * FROM services WHERE name = $1 and type = $2"
	return findOne(db, query, name, stype)
}

// FindByName a service by its name
func FindByName(db gorp.SqlExecutor, name string) (*sdk.Service, error) {
	query := "SELECT * FROM services WHERE name = $1"
	return findOne(db, query, name)
}

// FindByID a service by its id
func FindByID(db gorp.SqlExecutor, id int64) (*sdk.Service, error) {
	query := "SELECT * FROM services WHERE id = $1"
	return findOne(db, query, id)
}

// FindByType services by type
func FindByType(db gorp.SqlExecutor, t string) ([]sdk.Service, error) {
	if ss, ok := internalCache.getFromCache(t); ok {
		return ss, nil
	}
	query := `SELECT * FROM services WHERE type = $1`
	services, err := findAll(db, query, t)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to find dead services")
	}

	return services, nil
}

// All returns all registered services
func All(db gorp.SqlExecutor) ([]sdk.Service, error) {
	query := `SELECT * FROM services`
	services, err := findAll(db, query)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to find all services")
	}
	return services, nil
}

func findOne(db gorp.SqlExecutor, query string, args ...interface{}) (*sdk.Service, error) {
	sdb := service{}
	if err := db.SelectOne(&sdb, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, sdk.WithStack(err)
	}
	isValid, err := gorpmapping.CheckSignature(db, &sdb)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, sdk.WithStack(sdk.ErrCorruptedData)
	}
	if sdb.Name == "" {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &sdb.Service, nil
}

func findAll(db gorp.SqlExecutor, query string, args ...interface{}) ([]sdk.Service, error) {
	sdbs := []service{}
	if _, err := db.Select(&sdbs, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, sdk.WithStack(err)
	}
	ss := make([]sdk.Service, 0, len(sdbs))
	for i := 0; i < len(sdbs); i++ {
		isValid, err := gorpmapping.CheckSignature(db, &sdbs[i])
		if err != nil {
			log.Error("services.findAll> unable to load service id=%d: %v", sdbs[i].ID, err)
		}
		if !isValid {
			log.Error("services.findAll> unable to load service id=%d: %v", sdbs[i].ID, sdk.WithStack(sdk.ErrCorruptedData))
		}
		ss = append(ss, sdbs[i].Service)
	}
	return ss, nil
}

// Insert a service
func Insert(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service{
		Service: *s,
	}
	if err := gorpmapping.Encrypt(sdb.ClearJWT, &sdb.EncryptedJWT, []byte(sdb.Name)); err != nil {
		return err
	}
	sdb.ClearJWT = ""
	if err := gorpmapping.InsertAndSign(db, &sdb); err != nil {
		return err
	}
	*s = sdb.Service
	return nil
}

// Update a service
func Update(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service{
		Service: *s,
	}
	if err := gorpmapping.Encrypt(sdb.ClearJWT, &sdb.EncryptedJWT, []byte(sdb.Name)); err != nil {
		return err
	}
	sdb.ClearJWT = ""
	if err := gorpmapping.UpdatetAndSign(db, &sdb); err != nil {
		return err
	}
	*s = sdb.Service
	return nil
}

// Delete a service
func Delete(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service{
		Service: *s,
	}
	if _, err := db.Delete(&sdb); err != nil {
		return sdk.WrapError(err, "unable to delete service %s", s.Name)
	}
	return nil
}

// FindDeadServices returns services which haven't heart since th duration
func FindDeadServices(db gorp.SqlExecutor, t time.Duration) ([]sdk.Service, error) {
	query := `SELECT * FROM services WHERE last_heartbeat < $1`
	services, err := findAll(db, query, time.Now().Add(-1*t))
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to find dead services")
	}
	return services, nil
}

func LoadClearJWT(db gorp.SqlExecutor, id int64) (string, error) {
	query := gorpmapping.NewQuery("select encrypted_jwt from services where id = $1").Args(id)
	var encryptedJWT []byte
	if _, err := gorpmapping.Get(db, query, &encryptedJWT); err != nil {
		return "", err
	}
	var clearJWT string
	if err := gorpmapping.Decrypt(encryptedJWT, &clearJWT); err != nil {
		return "", err
	}
	return clearJWT, nil
}
