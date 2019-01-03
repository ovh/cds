package pipeline

import (
	"strconv"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

func keyBookJob(pbJobID int64) string {
	return cache.Key("book", "job", strconv.FormatInt(pbJobID, 10))
}

// BookPipelineBuildJob Book an action for a hatchery
func BookPipelineBuildJob(store cache.Store, pbJobID int64, hatchery *sdk.Service) (*sdk.Service, error) {
	k := keyBookJob(pbJobID)
	h := sdk.Service{}
	if !store.Get(k, &h) {
		// job not already booked, book it for 2 min
		store.SetWithTTL(k, hatchery, 120)
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrJobAlreadyBooked, "BookPipelineBuildJob> job %d already booked by %s (%d)", pbJobID, h.Name, h.ID)
}

func prepareSpawnInfos(pbJob *sdk.PipelineBuildJob, infos []sdk.SpawnInfo) error {
	now := time.Now()
	for _, info := range infos {
		pbJob.SpawnInfos = append(pbJob.SpawnInfos, sdk.SpawnInfo{
			APITime:    now,
			RemoteTime: info.RemoteTime,
			Message:    info.Message,
		})
	}
	return nil
}
