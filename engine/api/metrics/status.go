package metrics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
)

type computeGlobalNumbers struct {
	nbSrv       int
	nbOK        int
	nbAlerts    int
	nbWarn      int
	minInstance int
}

// MinInstances contains minInstance required for each uService
type MinInstances struct {
	TypeAPI           int
	TypeRepositories  int
	TypeVCS           int
	TypeHooks         int
	TypeHatchery      int
	TypeDBMigrate     int
	TypeElasticsearch int
}

var minInstances MinInstances

// ComputeGlobalStatus returns global status
func ComputeGlobalStatus(srvs []sdk.Service) sdk.MonitoringStatus {
	mStatus := sdk.MonitoringStatus{}

	var version string
	versionOk := true
	linesGlobal := []sdk.MonitoringStatusLine{}

	resume := map[string]computeGlobalNumbers{
		services.TypeAPI:           {minInstance: minInstances.TypeAPI},
		services.TypeRepositories:  {minInstance: minInstances.TypeRepositories},
		services.TypeVCS:           {minInstance: minInstances.TypeVCS},
		services.TypeHooks:         {minInstance: minInstances.TypeHooks},
		services.TypeHatchery:      {minInstance: minInstances.TypeHatchery},
		services.TypeDBMigrate:     {minInstance: minInstances.TypeDBMigrate},
		services.TypeElasticsearch: {minInstance: minInstances.TypeElasticsearch},
	}
	var nbg computeGlobalNumbers
	for _, s := range srvs {
		var nbOK, nbWarn, nbAlert int
		for i := range s.MonitoringStatus.Lines {
			l := s.MonitoringStatus.Lines[i]
			mStatus.Lines = append(mStatus.Lines, l)

			switch l.Status {
			case sdk.MonitoringStatusOK:
				nbOK++
			case sdk.MonitoringStatusWarn:
				nbWarn++
			default:
				nbAlert++
			}

			// services should have same version
			if strings.Contains(l.Component, "Version") {
				if version == "" {
					version = l.Value
				} else if version != l.Value && versionOk {
					versionOk = false
					linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
						Status:    sdk.MonitoringStatusWarn,
						Component: "Global/Version",
						Value:     fmt.Sprintf("%s vs %s", version, l.Value),
					})
				}
			}
		}

		t := resume[s.Type]
		t.nbOK += nbOK
		t.nbWarn += nbWarn
		t.nbAlerts += nbAlert
		t.nbSrv++
		resume[s.Type] = t

		nbg.nbOK += nbOK
		nbg.nbWarn += nbWarn
		nbg.nbAlerts += nbAlert
		nbg.nbSrv++
	}

	if versionOk {
		linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
			Status:    sdk.MonitoringStatusOK,
			Component: "Global/Version",
			Value:     version,
		})
	}

	linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
		Status:    computeGlobalStatusByNumbers(nbg),
		Component: "Global/Status",
		Value:     fmt.Sprintf("%d services", len(srvs)),
	})

	for stype, r := range resume {
		linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
			Status:    computeGlobalStatusByNumbers(r),
			Component: fmt.Sprintf("Global/%s", stype),
			Value:     fmt.Sprintf("%d", r.nbSrv),
		})
	}

	sort.Slice(linesGlobal, func(i, j int) bool {
		return linesGlobal[i].Component < linesGlobal[j].Component
	})

	mStatus.Lines = append(linesGlobal, mStatus.Lines...)
	return mStatus
}

func computeGlobalStatusByNumbers(s computeGlobalNumbers) string {
	r := sdk.MonitoringStatusOK
	if s.nbAlerts > 0 {
		r = sdk.MonitoringStatusAlert
	} else if s.nbWarn > 0 {
		r = sdk.MonitoringStatusWarn
	} else if s.nbSrv < s.minInstance {
		r = sdk.MonitoringStatusAlert
	}
	return r
}
