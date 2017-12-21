import {Component, Input} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Subscription} from 'rxjs/Subscription';
import {Pipeline, PipelineBuild, PipelineBuildJob, PipelineStatus} from '../../../model/pipeline.model';
import {Stage} from '../../../model/stage.model';
import {Job} from '../../../model/job.model';
import {Project} from '../../../model/project.model';
import {Application} from '../../../model/application.model';
import {DurationService} from '../../../shared/duration/duration.service';
import {NotificationService} from '../../../service/notification/notification.service';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {first} from 'rxjs/operators';


@Component({
    selector: 'app-pipeline-run-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class PipelineRunWorkflowComponent {

    @Input() previousBuild: PipelineBuild;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() project: Project;

    @Input('build')
    set build(data: PipelineBuild) {
        if (!data) {
            this.initData();
            return;
        }
        this.refreshBuild(data);
    }

    currentBuild: PipelineBuild;
    notificationSubscription: Subscription;
    previousStatus: string;
    pipelineStatusEnum = PipelineStatus;

    selectedPipJob: PipelineBuildJob;
    jobSelected: Job;
    mapStepStatus: { [key: string]: string };
    mapJobStatus: { [key: number]: string };
    mapJobProgression: { [key: number]: number };
    mapJobDuration: { [key: number]: string };
    manual = false;

    nextBuilds: Array<PipelineBuild>;

    constructor(private _durationService: DurationService, private _translate: TranslateService,
        private _notification: NotificationService, private _appPipService: ApplicationPipelineService) {
        this.initData();
    }

    initData(): void {
        delete this.currentBuild;
        delete this.selectedPipJob;
        delete this.jobSelected;
        this.mapStepStatus = {};
        this.mapJobStatus = {};
        this.mapJobProgression = {};
        this.mapJobDuration = {};
    }

    refreshBuild(data: PipelineBuild): void {
        let previousBuild = this.currentBuild;
        this.currentBuild = data;

        if (this.previousStatus && this.currentBuild && this.previousStatus === PipelineStatus.BUILDING &&
            this.previousBuild && this.previousBuild.id !== this.currentBuild.id &&
                this.currentBuild.status !== PipelineStatus.BUILDING) {
            this.handleNotification(this.currentBuild);
        }

        if (this.currentBuild) {
            this.previousStatus = this.currentBuild.status;
            if (this.currentBuild.status === PipelineStatus.SUCCESS) {
                this.getTriggeredPipeline();
            }
        }
        // Set selected job if needed or refresh step_status
        if (this.currentBuild.stages) {
            this.currentBuild.stages.forEach((s, sIndex) => {
                if (!this.manual && previousBuild && (!previousBuild.stages[sIndex].status ||
                    previousBuild.stages[sIndex].status === PipelineStatus.NEVER_BUILT) &&
                    (s.status === PipelineStatus.WAITING || s.status === PipelineStatus.BUILDING)) {
                  this.selectedJob(s.jobs[0], s);
                }
                if (s.builds) {
                    s.builds.forEach((pipJob, pjIndex) => {
                        // Update percent progression
                        if (pipJob.status === PipelineStatus.BUILDING) {
                            this.updateJobProgression(pipJob);
                        }
                        // Update duration
                        this.updateJobDuration(pipJob);

                        // Update map step status
                        if (pipJob.job.step_status) {
                            pipJob.job.step_status.forEach(ss => {
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

                        // Update spawninfo
                        if (this.selectedPipJob && this.selectedPipJob.id === pipJob.id) {
                            this.selectedPipJob.spawninfos = pipJob.spawninfos;
                        }

                        // Update status map for Job
                        this.mapJobStatus[pipJob.job.pipeline_action_id] = pipJob.status;
                    });
                }
            });
        }
    }

    handleNotification(pipelineBuild: PipelineBuild): void {
        switch (pipelineBuild.status) {
        case PipelineStatus.SUCCESS:
            this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_pipeline_success', {
                pipelineName: pipelineBuild.pipeline.name
            }), { icon: 'assets/images/checked.png' }).subscribe();
            break;
        case PipelineStatus.FAIL:
            this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_pipeline_failing', {
                pipelineName: pipelineBuild.pipeline.name
            }), { icon: 'assets/images/close.png' }).subscribe();
            break;
        }
    }

    updateJobDuration(pipJob: PipelineBuildJob): void {
        switch (pipJob.status) {
            case PipelineStatus.WAITING:
                if (pipJob.queued) {
                    this.mapJobDuration[pipJob.job.pipeline_action_id] =
                        'Queued ' + this._durationService.duration(new Date(pipJob.queued), new Date()) + ' ago';
                }
                break;
            case PipelineStatus.BUILDING:
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
            this.previousBuild.stages.forEach(s => {
                if (s.builds) {
                    s.builds.forEach(b => {
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
            s.builds.forEach(b => {
                if (b.job.pipeline_action_id === j.pipeline_action_id) {
                    this.selectedPipJob = b;
                }
            });
        }
    }

    getTriggeredPipeline(): void {
        this._appPipService.getTriggeredPipeline(
            this.project.key,
            this.currentBuild.application.name,
            this.currentBuild.pipeline.name,
            this.currentBuild.build_number)
            .pipe(first()).subscribe( builds => {
            this.nextBuilds = builds;
        });
    }
}
