package workflow

import (
	"fmt"
	"sort"
	"time"

	"github.com/fatih/structs"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// GetWorkflowRunEventData read channel to get elements to push
func GetWorkflowRunEventData(cError <-chan error, cEvent <-chan interface{}) ([]sdk.WorkflowRun, []sdk.WorkflowNodeRun, []sdk.WorkflowNodeJobRun, error) {
	wrs := []sdk.WorkflowRun{}
	wnrs := []sdk.WorkflowNodeRun{}
	wnjrs := []sdk.WorkflowNodeJobRun{}
	var err error

	for {
		select {
		case e, has := <-cError:
			if e != nil {
				err = sdk.WrapError(e, "GetWorkflowRunEventData> Error received")
			}

			if !has {
				return wrs, wnrs, wnjrs, err
			}
		case w, has := <-cEvent:
			if !has {
				return wrs, wnrs, wnjrs, err
			}
			switch x := w.(type) {
			case sdk.WorkflowNodeJobRun:
				wnjrs = append(wnjrs, x)
			case sdk.WorkflowNodeRun:
				wnrs = append(wnrs, x)
			case sdk.WorkflowRun:
				wrs = append(wrs, x)
			default:
				log.Warning("GetWorkflowRunEventData> unknown type %T", w)
			}
		}
	}
}

// SendEvent Send event on workflow run
func SendEvent(db gorp.SqlExecutor, wrs []sdk.WorkflowRun, wnrs []sdk.WorkflowNodeRun, wnjrs []sdk.WorkflowNodeJobRun, key string) {
	for _, wr := range wrs {
		event.PublishWorkflowRun(wr, key)
	}
	for _, wnr := range wnrs {
		wr, errWR := LoadRunByID(db, wnr.WorkflowRunID, false)
		if errWR != nil {
			log.Warning("SendEvent.workflow> Cannot load workflow run %d: %s", wnr.WorkflowRunID, errWR)
			continue
		}

		var previousNodeRun sdk.WorkflowNodeRun
		if wnr.SubNumber > 0 {
			previousNodeRun = wnr
		} else {
			// Load previous run on current node
			node := wr.Workflow.GetNode(wnr.WorkflowNodeID)
			if node != nil {
				var errN error
				previousNodeRun, errN = PreviousNodeRun(db, wnr, *node, wr.WorkflowID)
				if errN != nil {
					log.Debug("SendEvent.workflow> Cannot load previous node run: %s", errN)
				}
			} else {
				log.Warning("SendEvent.workflow > Unable to find node %d in workflow", wnr.WorkflowNodeID)
			}
		}

		event.PublishWorkflowNodeRun(db, wnr, *wr, previousNodeRun, key)
	}
	for _, wnjr := range wnjrs {
		event.PublishWorkflowNodeJobRun(wnjr)
	}
}

func resyncCommitStatus(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun) error {
	log.Debug("resyncCommitStatus> %s %d.%d", wr.Workflow.Name, wr.Number, wr.LastSubNumber)
	for nodeID, nodeRuns := range wr.WorkflowNodeRuns {
		sort.Slice(nodeRuns, func(i, j int) bool {
			return nodeRuns[i].SubNumber >= nodeRuns[j].SubNumber
		})

		nodeRun := nodeRuns[0]
		if !sdk.StatusIsTerminated(nodeRun.Status) {
			continue
		}

		node := wr.Workflow.GetNode(nodeID)
		if node != nil && node.Context != nil && node.Context.Application != nil && node.Context.Application.VCSServer != "" && node.Context.Application.RepositoryFullname != "" {
			vcsServer := repositoriesmanager.GetProjectVCSServer(p, node.Context.Application.VCSServer)
			if vcsServer == nil {
				return nil
			}

			//Get the RepositoriesManager Client
			client, errClient := repositoriesmanager.AuthorizedClient(db, store, vcsServer)
			if errClient != nil {
				return sdk.WrapError(errClient, "resyncCommitStatus> Cannot get client")
			}

			statuses, errStatuses := client.ListStatuses(node.Context.Application.RepositoryFullname, nodeRun.VCSHash)
			if errStatuses != nil {
				return sdk.WrapError(errStatuses, "resyncCommitStatus> Cannot get statuses")
			}

			var statusFound *sdk.VCSCommitStatus
			expected := sdk.VCSCommitStatusDescription(sdk.EventWorkflowNodeRun{
				ProjectKey:   p.Key,
				WorkflowName: wr.Workflow.Name,
				NodeName:     node.Name,
			})

			var sendEvent = func() error {
				log.Debug("Resync status for node run %d", nodeRun.ID)
				var eventWNR = sdk.EventWorkflowNodeRun{
					ID:             nodeRun.ID,
					Number:         nodeRun.Number,
					SubNumber:      nodeRun.SubNumber,
					Status:         nodeRun.Status,
					Start:          nodeRun.Start.Unix(),
					Done:           nodeRun.Done.Unix(),
					ProjectKey:     p.Key,
					Manual:         nodeRun.Manual,
					HookEvent:      nodeRun.HookEvent,
					Payload:        nodeRun.Payload,
					SourceNodeRuns: nodeRun.SourceNodeRuns,
					WorkflowName:   wr.Workflow.Name,
					Hash:           nodeRun.VCSHash,
					BranchName:     nodeRun.VCSBranch,
				}

				node := wr.Workflow.GetNode(nodeRun.WorkflowNodeID)
				if node != nil {
					eventWNR.PipelineName = node.Pipeline.Name
					eventWNR.NodeName = node.Name
				}
				if node.Context != nil {
					if node.Context.Application != nil {
						eventWNR.ApplicationName = node.Context.Application.Name
						eventWNR.RepositoryManagerName = node.Context.Application.VCSServer
						eventWNR.RepositoryFullName = node.Context.Application.RepositoryFullname
					}
					if node.Context.Environment != nil {
						eventWNR.EnvironmentName = node.Context.Environment.Name
					}
				}

				evt := sdk.Event{
					EventType: fmt.Sprintf("%T", eventWNR),
					Payload:   structs.Map(eventWNR),
					Timestamp: time.Now(),
				}
				if err := client.SetStatus(evt); err != nil {
					repositoriesmanager.RetryEvent(&evt, err, store)
					return fmt.Errorf("resyncCommitStatus> err:%s", err)
				}
				return nil
			}

			for i, status := range statuses {
				if status.Decription == expected {
					statusFound = &statuses[i]
					break
				}
			}

			if statusFound == nil {
				if err := sendEvent(); err != nil {
					log.Error("resyncCommitStatus> Error sending status: %v", err)
				}
				continue
			}

			if statusFound.State == sdk.StatusBuilding.String() {
				if err := sendEvent(); err != nil {
					log.Error("resyncCommitStatus> Error sending status: %v", err)
				}
				continue
			}

			switch statusFound.State {
			case sdk.StatusSuccess.String():
				switch nodeRun.Status {
				case sdk.StatusSuccess.String():
					continue
				default:
					if err := sendEvent(); err != nil {
						log.Error("resyncCommitStatus> Error sending status: %v", err)
					}
					continue
				}

			case sdk.StatusFail.String():
				switch nodeRun.Status {
				case sdk.StatusFail.String():
					continue
				default:
					if err := sendEvent(); err != nil {
						log.Error("resyncCommitStatus> Error sending status: %v", err)
					}
					continue
				}

			case sdk.StatusSkipped.String():
				switch nodeRun.Status {
				case sdk.StatusDisabled.String(), sdk.StatusNeverBuilt.String(), sdk.StatusSkipped.String():
					continue
				default:
					if err := sendEvent(); err != nil {
						log.Error("resyncCommitStatus> Error sending status: %v", err)
					}
					continue
				}
			}
		}
	}
	return nil
}
