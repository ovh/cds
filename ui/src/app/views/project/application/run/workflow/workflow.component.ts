import {Component, Input, OnInit, OnDestroy, NgZone} from '@angular/core';
import {PipelineBuild, PipelineBuildJob, Pipeline} from '../../../../../model/pipeline.model';
import {Stage} from '../../../../../model/stage.model';
import {Job, StepStatus} from '../../../../../model/job.model';
import {Subscription} from 'rxjs/Rx';
import {CDSWorker} from '../../../../../shared/worker/worker';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';

@Component({
    selector: 'app-pipeline-run-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
export class PipelineRunWorkflowComponent implements OnInit, OnDestroy {

    @Input() buildWorker: CDSWorker;
    @Input() previousBuild: PipelineBuild;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() project: Project;

    currentBuild: PipelineBuild;
    selectedPipJob: PipelineBuildJob;
    jobSelected: Job;
    mapStepStatus: {[key: string]: string} = {};
    mapJobStatus: {[key: number]: string} = {};

    // Allow angular update from work started outside angular context
    zone: NgZone;

    workerSubscription: Subscription;

    constructor() {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnDestroy(): void {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    ngOnInit(): void {
        this.workerSubscription = this.buildWorker.response().subscribe(msg => {
            if (msg.data) {
                this.zone.run(() => {
                    this.currentBuild = JSON.parse(msg.data);
                    // Set selected job if needed or refresh step_status

                    if (this.currentBuild.stages) {
                        this.currentBuild.stages.forEach(s => {
                            if (s.builds) {
                                s.builds.forEach(pipJob => {
                                    if (this.selectedPipJob) {
                                        this.selectedPipJob.job.step_status = pipJob.job.step_status;

                                    }

                                    if (this.jobSelected && !this.selectedPipJob
                                        && pipJob.job.pipeline_action_id === this.jobSelected.pipeline_action_id) {
                                        this.selectedJob(this.jobSelected, s);
                                    }

                                    // Update map step status
                                    if (pipJob.job.step_status) {
                                        pipJob.job.step_status.forEach( ss => {
                                            this.mapStepStatus[pipJob.job.pipeline_action_id + '-' + ss.step_order] = ss.status;
                                        });
                                    }

                                    // Update status map for JOb
                                    this.mapJobStatus[pipJob.job.pipeline_action_id] = pipJob.status;
                                });
                            }
                        });
                    }
                });
            }
        });
    }

    selectedJob(j: Job, s: Stage): void {
        this.jobSelected = j;
        if (s.builds) {
            s.builds.forEach( b => {
                if (b.job.pipeline_action_id === j.pipeline_action_id) {
                    this.selectedPipJob = b;
                }
            });
        }
    }
}
