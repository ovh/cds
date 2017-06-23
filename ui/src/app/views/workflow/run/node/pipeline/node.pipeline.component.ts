import {Component, Input} from '@angular/core';
import {WorkflowNodeJobRun, WorkflowNodeRun} from '../../../../../model/workflow.run.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {Project} from '../../../../../model/project.model';

declare var Duration: any;

@Component({
    selector: 'app-node-run-pipeline',
    templateUrl: './pipeline.html',
    styleUrls: ['./pipeline.scss']
})
export class WorkflowRunNodePipelineComponent {

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
    mapStepStatus: Map<string, string> = new Map<string, string>();

    previousStatus: string;

    constructor() { }

    selectedJob(rj: WorkflowNodeJobRun): void {
        this.selectedRunJob = rj;
    }

    refreshNodeRun(data: WorkflowNodeRun): void {
        this.nodeRun = data;

        /*
        if (this.previousStatus && this.nodeRun && this.previousStatus === PipelineStatus.BUILDING &&
            this.previousBuild && this.previousBuild.id !== this.currentBuild.id &&
            this.nodeRun.status !== PipelineStatus.BUILDING) {
            this.handleNotification(this.currentBuild);
        }
        */

        if (this.nodeRun) {
            this.previousStatus = this.nodeRun.status;
            if (this.nodeRun.status === PipelineStatus.SUCCESS) {
                this.getTriggeredNodeRun();
            }
        }
        // Set selected job if needed or refresh step_status
        if (this.nodeRun.stages) {
            this.nodeRun.stages.forEach((s, sIndex) => {

                if (s.run_jobs) {
                    s.run_jobs.forEach((rj, rjIndex) => {
                        // Update percent progression
                        if (rj.status === PipelineStatus.BUILDING) {
                            // this.updateJobProgression(rj);
                        }
                        // Update duration
                        // this.updateJobDuration(rj);

                        // Update map step status
                        if (rj.job.step_status) {
                            rj.job.step_status.forEach(ss => {
                                this.mapStepStatus[rj.job.pipeline_action_id + '-' + ss.step_order] = ss.status;
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
                       this.jobTime.set(rj.id, new Duration(rj.queued_seconds + 's'));
                       if (rj.job.step_status) {
                           rj.job.step_status.forEach(ss => {
                               this.mapStepStatus.set(rj.job.pipeline_action_id + '-' + ss.step_order, ss.status);
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
