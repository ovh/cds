package venom

import (
	"sort"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
)

func initBars(detailsLevel string, bars map[string]*pb.ProgressBar) *pb.Pool {
	var pool *pb.Pool
	var pbbars []*pb.ProgressBar
	if detailsLevel != DetailsLow {
		// sort bars pool
		var keys []string
		for k := range bars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			pbbars = append(pbbars, bars[k])
		}
		var errs error
		pool, errs = pb.StartPool(pbbars...)
		if errs != nil {
			log.Errorf("Error while prepare details bars: %s", errs)
			pool = nil
		}
	}
	return pool
}

func endBars(detailsLevel string, pool *pb.Pool) {
	if detailsLevel != DetailsLow && pool != nil {
		if err := pool.Stop(); err != nil {
			log.Errorf("Error while closing pool progress bar: %s", err)
		}
	}
}
