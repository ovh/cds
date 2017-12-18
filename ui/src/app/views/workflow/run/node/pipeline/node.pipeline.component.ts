import {Component, Input, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {WorkflowNodeJobRun, WorkflowNodeRun} from '../../../../../model/workflow.run.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {Project} from '../../../../../model/project.model';
import {Job, StepStatus} from '../../../../../model/job.model';
import {DurationService} from '../../../../../shared/duration/duration.service';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-node-run-pipeline',
    templateUrl: './pipeline.html',
    styleUrls: ['./pipeline.scss']
})
export class WorkflowRunNodePipelineComponent implements OnInit {

    nodeRun: WorkflowNodeRun;
    jobTime: Map<number, string>;

    @Input() workflowName: string;
    @Input() project: Project;
    @Input('run')
    set run(data: WorkflowNodeRun) {
         this.refreshNodeRun(data);
         this.updateTime();
    }

    pipelineStatusEnum = PipelineStatus;
    selectedRunJob: WorkflowNodeJobRun;
    mapJobStatus: Map<number, string> = new Map<number, string>();
    mapStepStatus: Map<string, StepStatus> = new Map<string, StepStatus>();

    previousStatus: string;
    manual = false;

    constructor(private _durationService: DurationService, private _route: ActivatedRoute, private _router: Router) { }

    ngOnInit() {
      if (!this._route.snapshot.queryParams['actionId'] && this._route.snapshot.queryParams['stageId']) {
        this.selectedStage(parseInt(this._route.snapshot.queryParams['stageId'], 10));
      } else if (this._route.snapshot.queryParams['actionId']) {
        let job = new Job();
        job.pipeline_action_id = parseInt(this._route.snapshot.queryParams['actionId'], 10);
        this.manual = true;
        this.selectedJob(job);
      }
    }

    selectedJobManual(j: Job) {
      let queryParams = cloneDeep(this._route.snapshot.queryParams);
      queryParams['stageId'] = null;
      queryParams['actionId'] = null;
      queryParams['stepOrder'] = null;
      queryParams['line'] = null;
      this.manual = true;

      this._router.navigate(['.'], { relativeTo: this._route, queryParams, fragment: null });
      this.selectedJob(j);
    }

    selectedStage(stageId: number) {
      let stage = this.nodeRun.stages.find((st) => st.id === stageId);

      if (stage && Array.isArray(stage.run_jobs) && stage.run_jobs.length) {
        this.selectedRunJob = stage.run_jobs[0];
      }
    }

    selectedJob(j: Job): void {
        this.nodeRun.stages.forEach(s => {
            if (s.run_jobs) {
                let runJob = s.run_jobs.find(rj => rj.job.pipeline_action_id === j.pipeline_action_id);
                if (runJob) {
                    this.selectedRunJob = runJob;
                }
            }
        });
    }

    refreshNodeRun(data: WorkflowNodeRun): void {
        let previousRun = this.nodeRun;
        this.nodeRun = data;

        if (this.nodeRun) {
            this.previousStatus = this.nodeRun.status;
            if (this.nodeRun.status === PipelineStatus.SUCCESS) {
                this.getTriggeredNodeRun();
            }
        }
        // Set selected job if needed or refresh step_status
        if (this.nodeRun.stages) {
            this.nodeRun.stages.forEach((s, sIndex) => {
                if (!this.manual && previousRun && (!previousRun.stages[sIndex].status ||
                    previousRun.stages[sIndex].status === PipelineStatus.NEVER_BUILT) &&
                    (s.status === PipelineStatus.WAITING || s.status === PipelineStatus.BUILDING)) {
                  this.selectedJob(s.jobs[0]);
                }
                if (s.run_jobs) {
                    s.run_jobs.forEach((rj, rjIndex) => {
                        // Update job status
                        this.mapJobStatus.set(rj.job.pipeline_action_id, rj.status);

                        // Update map step status
                        if (rj.job.step_status) {
                            rj.job.step_status.forEach(ss => {
                                this.mapStepStatus[rj.job.pipeline_action_id + '-' + ss.step_order] = ss;
                            });
                        }

                        // Select temp job
                        if (!this.selectedRunJob && sIndex === 0 && rjIndex === 0) {
                            this.selectedRunJob = rj;
                        }

                        // Update spawninfo
                        if (this.selectedRunJob && this.selectedRunJob.id === rj.id) {
                            this.selectedRunJob.spawninfos = rj.spawninfos;
                        }
                    });
                }
            });
        }
    }

    updateTime(): void {
        this.jobTime = new Map<number, string>();
        if (this.nodeRun.stages) {
            this.nodeRun.stages.forEach(s => {

               if (s.run_jobs) {
                   s.run_jobs.forEach(rj => {
                       switch (rj.status) {
                           case this.pipelineStatusEnum.WAITING:
                               this.jobTime.set(rj.job.pipeline_action_id, this._durationService.duration(new Date(rj.queued), new Date()));
                               break;
                           case this.pipelineStatusEnum.BUILDING:
                               this.jobTime.set(rj.job.pipeline_action_id, this._durationService.duration(new Date(rj.start), new Date()));
                               break;
                           case this.pipelineStatusEnum.SUCCESS:
                           case this.pipelineStatusEnum.FAIL:
                               this.jobTime.set(rj.job.pipeline_action_id,
                                   this._durationService.duration( new Date(rj.start), new Date(rj.done) ));
                               break;
                       }

                       if (rj.job.step_status) {
                           rj.job.step_status.forEach(ss => {
                               this.mapStepStatus.set(rj.job.pipeline_action_id + '-' + ss.step_order, ss);
                           });
                       }
                   });
               }
            });
        }
     }

    getTriggeredNodeRun() {

    }
}
