package gorpmapping

import (
	"github.com/ovh/cds/engine/gorpmapper"
)

var Mapper *gorpmapper.Mapper

func init() {
	Mapper = gorpmapper.New()
}
