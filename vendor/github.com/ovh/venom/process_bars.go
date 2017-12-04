package venom

import (
	"sort"

	log "github.com/sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
)

func (v *Venom) initBars() *pb.Pool {
	var pool *pb.Pool
	var pbbars []*pb.ProgressBar
	// sort bars pool
	var keys []string
	for k := range v.outputProgressBar {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		pbbars = append(pbbars, v.outputProgressBar[k])
	}
	var errs error
	pool, errs = pb.StartPool(pbbars...)
	if errs != nil {
		log.Errorf("Error while prepare details bars: %s", errs)
		pool = nil
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
