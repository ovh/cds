package gorpmapping

import "github.com/ovh/cds/engine/gorpmapper"

func New(target interface{}, name string, autoIncrement bool, keys ...string) gorpmapper.TableMapping {
	return Mapper.NewTableMapping(target, name, autoIncrement, keys...)
}
