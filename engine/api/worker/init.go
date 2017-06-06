package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
)

//Initialize init the package
func Initialize(c context.Context, DBFunc func() *gorp.DbMap) error {
	go CheckHeartbeat(c, DBFunc)
	go ModelCapabilititiesCacheLoader(c, 10*time.Second, DBFunc)
	return nil
}
