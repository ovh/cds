package workflowv3

import (
	"sort"
	"time"

	"github.com/ovh/cds/sdk"
)

func ConvertRun(wr *sdk.WorkflowRun, isFullExport bool) WorkflowRun {
	res := NewWorkflowRun()

	res.Number = wr.Number

	info := sdk.SpawnMsg{
		ID:   sdk.MsgWorkflowV3Preview.ID,
		Type: sdk.MsgWorkflowV3Preview.Type,
	}
	res.Infos = append(wr.Infos, sdk.WorkflowRunInfo{
		APITime:     time.Now(),
		Message:     info,
		Type:        info.Type,
		UserMessage: info.DefaultUserMessage(),
	})
	res.Resources.Workflow = Convert(wr.Workflow, isFullExport)

	// Build integration resources
	extDep, _ := res.Resources.Workflow.Validate()
	for _, i := range wr.Workflow.Integrations {
		if _, ok := wr.Workflow.ProjectIntegrations[i.ProjectIntegrationID]; ok && i.ProjectIntegration.Model.ArtifactManager {
			wr.Workflow.ProjectIntegrations[i.ProjectIntegrationID] = i.ProjectIntegration
		}
	}
	for _, i := range extDep.Integrations {
		for _, pi := range wr.Workflow.ProjectIntegrations {
			if pi.Name == i {
				res.Resources.Integrations = append(res.Resources.Integrations, pi)
			}
		}
		for _, wi := range wr.Workflow.Integrations {
			if wi.ProjectIntegration.Name == i {
				res.Resources.Integrations = append(res.Resources.Integrations, wi.ProjectIntegration)
			}
		}
	}

	// Set job runs
	for _, execs := range wr.WorkflowNodeRuns {
		for _, exec := range execs {
			node := wr.Workflow.WorkflowData.NodeByID(exec.WorkflowNodeID)
			for _, s := range exec.Stages {
				for _, j := range s.RunJobs {
					jID := computeJobUniqueID(node.Name, s.Name, j.Job.Action.Name, j.Job.Action.ID)
					var jName string
					for name, job := range res.Resources.Workflow.Jobs {
						if job.ID == jID {
							jName = name
							break
						}
					}
					if jName == "" {
						continue
					}

					if _, ok := res.JobRuns[jName]; !ok {
						res.JobRuns[jName] = nil
					}

					sStatus := make([]StepStatus, len(j.Job.StepStatus))
					for i := range j.Job.StepStatus {
						sStatus[i] = StepStatus{
							StepOrder: int64(j.Job.StepStatus[i].StepOrder),
							Status:    j.Job.StepStatus[i].Status,
							Start:     j.Job.StepStatus[i].Start,
							Done:      j.Job.StepStatus[i].Done,
						}
					}
					sort.Slice(sStatus, func(i, j int) bool { return sStatus[i].StepOrder < sStatus[j].StepOrder })

					res.JobRuns[jName] = append(res.JobRuns[jName], JobRun{
						Status:               j.Status,
						SubNumber:            exec.SubNumber,
						StepStatus:           sStatus,
						WorkflowNodeRunID:    j.WorkflowNodeRunID,
						WorkflowNodeJobRunID: j.ID,
					})
				}
			}
		}
	}
	for k := range res.JobRuns {
		sort.Slice(res.JobRuns[k], func(i, j int) bool { return res.JobRuns[k][i].SubNumber > res.JobRuns[k][j].SubNumber })
	}

	return res
}
