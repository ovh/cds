package worker

import (
	"context"

	"github.com/go-gorp/gorp"
)

//Initialize init the package
func Initialize(c context.Context, DBFunc func() *gorp.DbMap) error {
	go CheckHeartbeat(c, DBFunc)
	go ModelCapabilititiesCacheLoader(c, 5, DBFunc)
	return nil
}
