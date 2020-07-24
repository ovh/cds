package gorpmapping

import "github.com/ovh/cds/engine/gorpmapper"

func Register(ms ...gorpmapper.TableMapping) {
	Mapper.Register(ms...)
}
