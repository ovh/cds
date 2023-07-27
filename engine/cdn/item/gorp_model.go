package item

import (
	"encoding/json"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InitDBMapping(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(cdnItemDB{}, "item", false, "id"))
}

type cdnItemDB struct {
	gorpmapper.SignedEntity
	sdk.CDNItem
	APIRefDB json.RawMessage `json:"-" db:"api_ref"`
}

func (c cdnItemDB) ToCDSItem() (sdk.CDNItem, error) {
	item := c.CDNItem
	switch item.Type {
	case sdk.CDNTypeItemServiceLog, sdk.CDNTypeItemStepLog:
		var apiRef sdk.CDNLogAPIRef
		if err := sdk.JSONUnmarshal(c.APIRefDB, &apiRef); err != nil {
			return item, sdk.WithStack(err)
		}
		item.APIRef = &apiRef
	case sdk.CDNTypeItemJobStepLog:
		var apiRef sdk.CDNLogAPIRefV2
		if err := sdk.JSONUnmarshal(c.APIRefDB, &apiRef); err != nil {
			return item, sdk.WithStack(err)
		}
		item.APIRef = &apiRef
	case sdk.CDNTypeItemRunResult:
		var apiRef sdk.CDNRunResultAPIRef
		if err := sdk.JSONUnmarshal(c.APIRefDB, &apiRef); err != nil {
			return item, sdk.WithStack(err)
		}
		item.APIRef = &apiRef
	case sdk.CDNTypeItemWorkerCache:
		var apiRef sdk.CDNWorkerCacheAPIRef
		if err := sdk.JSONUnmarshal(c.APIRefDB, &apiRef); err != nil {
			return item, sdk.WithStack(err)
		}
		item.APIRef = &apiRef
	}
	return item, nil
}

func toItemDB(c sdk.CDNItem) cdnItemDB {
	bts, _ := json.Marshal(c.APIRef)
	raw := json.RawMessage(bts)
	return cdnItemDB{CDNItem: c, APIRefDB: raw}
}

func (c cdnItemDB) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{c.ID, c.APIRefHash, c.Type} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.APIRefHash}}{{.Type}}",
	}
}
