import {Component, Input, OnInit, OnDestroy, NgZone} from '@angular/core';
import {PipelineBuild, PipelineBuildJob, Pipeline} from '../../../model/pipeline.model';
import {Stage} from '../../../model/stage.model';
import {Job} from '../../../model/job.model';
import {Subscription} from 'rxjs/Rx';
import {CDSWorker} from '../../../shared/worker/worker';
import {Project} from '../../../model/project.model';
import {Application} from '../../../model/application.model';
import {DurationService} from '../../../shared/duration/duration.service';

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
    mapJobProgression: {[key: number]: number} = {};
    mapJobDuration: {[key: number]: string} = {};

    // Allow angular update from work started outside angular context
    zone: NgZone;

    workerSubscription: Subscription;

    constructor(private _durationService: DurationService) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnDestroy(): void {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    ngOnInit(): void {
        this.workerSubscription = this.buildWorker.response().subscribe(msg => {
            if (msg) {
                this.zone.run(() => {
                    this.currentBuild = JSON.parse(msg);

                    // Set selected job if needed or refresh step_status
                    if (this.currentBuild.stages) {
                        this.currentBuild.stages.forEach( (s, sIndex) => {

                            if (s.builds) {
                                s.builds.forEach( (pipJob, pjIndex) => {
                                    // Update percent progression
                                    if (pipJob.status === 'Building') {
                                        this.updateJobProgression(pipJob);
                                    }
                                    // Update duration
                                    this.updateJobDuration(pipJob);

                                    // Update map step status
                                    if (pipJob.job.step_status) {
                                        pipJob.job.step_status.forEach( ss => {
                                            this.mapStepStatus[pipJob.job.pipeline_action_id + '-' + ss.step_order] = ss.status;
                                        });
                                    }

                                    // Select temp job
                                    if (!this.jobSelected && sIndex === 0 && pjIndex === 0) {
                                        this.jobSelected = pipJob.job;
                                    }
                                    // Simulate click on job
                                    if (this.jobSelected && !this.selectedPipJob &&
                                        pipJob.job.pipeline_action_id === this.jobSelected.pipeline_action_id) {
                                        this.selectedJob(this.jobSelected, s);
                                    }

                                    // Update status map for Job
                                    this.mapJobStatus[pipJob.job.pipeline_action_id] = pipJob.status;
                                });
                            }
                        });
                    }
                });
            }
        });
    }

    updateJobDuration(pipJob: PipelineBuildJob): void {
        switch (pipJob.status) {
            case 'Waiting':
                if (pipJob.queued) {
                    this.mapJobDuration[pipJob.job.pipeline_action_id] =
                        'Queued ' + this._durationService.duration(new Date(pipJob.queued), new Date()) + ' ago';
                }
                break;
            case 'Building':
                if (pipJob.start) {
                    this.mapJobDuration[pipJob.job.pipeline_action_id] =
                        this._durationService.duration(new Date(pipJob.start), new Date());
                }
                break;
            default:
                if (pipJob.start && pipJob.done) {
                    this.mapJobDuration[pipJob.job.pipeline_action_id] =
                        this._durationService.duration(new Date(pipJob.start), new Date(pipJob.done));
                }
        }
    }

    /**
     * Update map with job progression
     * @param pipJob
     */
    updateJobProgression(pipJob: PipelineBuildJob): void {
        if (!this.previousBuild) {
            return;
        }
        if (this.previousBuild.stages) {
            this.previousBuild.stages.forEach( s => {
               if (s.builds) {
                   s.builds.forEach( b => {
                      if (b.job.pipeline_action_id !== pipJob.job.pipeline_action_id) {
                          return;
                      }
                      let previousTime = new Date(b.done).getTime() - new Date(b.start).getTime();
                      let currentTime = new Date().getTime() - new Date(pipJob.start).getTime();
                      let percent = Math.floor(100 * currentTime / previousTime);
                      if (percent > 99) {
                          percent = 99;
                      }
                      this.mapJobProgression[b.job.pipeline_action_id] = percent;
                   });
               }
            });
        }
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
