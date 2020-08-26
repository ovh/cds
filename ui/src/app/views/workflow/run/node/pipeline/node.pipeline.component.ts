import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Select, Store } from '@ngxs/store';
import { Job } from 'app/model/job.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Stage } from 'app/model/stage.model';
import { WorkflowNodeJobRun, WorkflowNodeRun } from 'app/model/workflow.run.model';
import { FeatureService } from 'app/service/feature/feature.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { AddFeatureResult, FeaturePayload } from 'app/store/feature.action';
import { FeatureResult } from 'app/store/feature.state';
import { ProjectState } from 'app/store/project.state';
import { SelectWorkflowNodeRunJob } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-node-run-pipeline',
    templateUrl: './pipeline.html',
    styleUrls: ['./pipeline.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunNodePipelineComponent implements OnInit, OnDestroy {

    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;

    @Select(WorkflowState.getSelectedWorkflowNodeJobRun()) nodeJobRun$: Observable<WorkflowNodeJobRun>;
    nodeJobRunSubs: Subscription;


    workflowName: string;
    project: Project;

    // Pipeline data
    stages: Array<Stage>;
    jobTime: Map<number, string>;
    mapJobStatus: Map<number, { status: string, warnings: number, start: string, done: string }>
        = new Map<number, { status: string, warnings: number, start: string, done: string }>();

    queryParamsSub: Subscription;
    pipelineStatusEnum = PipelineStatus;

    currentNodeRunID: number;
    currentNodeRunNum: number;
    currentJob: Job;

    displayServiceLogs = false;
    durationIntervalID: number;

    constructor(
        private _durationService: DurationService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _featureService: FeatureService
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.workflowName = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowRun.workflow.name;
    }

    ngOnInit() {
        this._featureService.isEnabled('cdn-job-logs', { 'project_key': this.project.key }).subscribe(f => {
            this._store.dispatch(new AddFeatureResult(<FeaturePayload>{
                key: f.name,
                result: <FeatureResult>{
                    paramString: this.project.key,
                    enabled: f.enabled
                }
            }));
        });
        this.nodeJobRunSubs = this.nodeJobRun$.subscribe(rj => {
            if (!rj && !this.currentJob) {
                return;
            }
            if (!rj) {
                delete this.currentJob;
                this._cd.markForCheck();
                return;
            }
            if (rj && this.currentJob && rj.job.pipeline_action_id === this.currentJob.pipeline_action_id) {
                return;
            }
            this.currentJob = rj.job;
            this._cd.markForCheck();
        });
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
            if (!nr) {
                return;
            }
            if (this.currentNodeRunID !== nr.id) {
                this.currentNodeRunID = nr.id;
                this.currentNodeRunNum = nr.num;
                this.stages = nr.stages;
                this.refreshNodeRun(nr);
                this.deleteInterval();
                this.updateTime();
                this.durationIntervalID = window.setInterval(() => {
                    this.updateTime();
                }, 5000);
                this._cd.markForCheck();
            } else {
                if (this.refreshNodeRun(nr)) {
                    this._cd.markForCheck();
                }
            }
        });
    }

    selectedJobManual(jobID: number) {
        if (!this.mapJobStatus.has(jobID)) {
            return;
        }
        let queryParams = cloneDeep(this._route.snapshot.queryParams);
        queryParams['stageId'] = null;
        queryParams['actionId'] = null;
        queryParams['stepOrder'] = null;
        queryParams['line'] = null;
        this._router.navigate(['.'], { relativeTo: this._route, queryParams, fragment: null });
        this.selectJob(jobID);
    }

    selectJob(jobID: number): void {
        if (this.currentJob && jobID === this.currentJob.pipeline_action_id) {
            return;
        }
        this._store.dispatch(new SelectWorkflowNodeRunJob({ jobID: jobID }));
    }

    refreshNodeRun(data: WorkflowNodeRun): boolean {
        let refresh = false;
        let currentNodeJobRun = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeJobRun;

        if (data.stages) {
            data.stages.forEach((s, sIndex) => {
                // Test Job status
                if (s.run_jobs) {
                    s.run_jobs.forEach((rj, rjIndex) => {

                        let warnings = 0;
                        // compute warning
                        if (rj.job.step_status) {
                            rj.job.step_status.forEach(ss => {
                                if (ss.status === PipelineStatus.FAIL && rj.job.action.actions[ss.step_order] &&
                                    rj.job.action.actions[ss.step_order].optional) {
                                    warnings++;
                                }
                            });
                        }

                        // Update job status
                        let jobStatusItem = this.mapJobStatus.get(rj.job.pipeline_action_id);
                        if (!jobStatusItem || jobStatusItem.status !== rj.status) {
                            refresh = true;
                            this.mapJobStatus.set(rj.job.pipeline_action_id,
                                { status: rj.status, warnings, start: rj.start, done: rj.done });
                        }

                        if (!currentNodeJobRun && sIndex === 0 && rjIndex === 0 && !this._route.snapshot.queryParams['actionId']) {
                            refresh = true;
                            this.selectJob(s.jobs[0].pipeline_action_id);
                        } else if (currentNodeJobRun && currentNodeJobRun.job.pipeline_action_id === this.currentJob.pipeline_action_id) {
                            this.selectJob(this.currentJob.pipeline_action_id);
                        } else if (this._route.snapshot.queryParams['actionId'] &&
                            this._route.snapshot.queryParams['actionId'] === rj.job.pipeline_action_id.toString()) {
                            this.selectJob(rj.job.pipeline_action_id);
                        }
                    });
                }
            });
        }
        return refresh;
    }

    /**
     * Update job time
     */
    updateTime(): void {
        if (!this.mapJobStatus || this.mapJobStatus.size === 0) {
            return;
        }
        if (!this.jobTime) {
            this.jobTime = new Map<number, string>();
        }
        let stillRunning = false;
        let refresh = false;
        this.mapJobStatus.forEach((v, k) => {
            switch (v.status) {
                case this.pipelineStatusEnum.WAITING:
                case this.pipelineStatusEnum.BUILDING:
                    refresh = true;
                    stillRunning = true;
                    this.jobTime.set(k,
                        this._durationService.duration(new Date(v.start), new Date()));
                    break;
                case this.pipelineStatusEnum.SUCCESS:
                case this.pipelineStatusEnum.FAIL:
                case this.pipelineStatusEnum.STOPPED:
                    let dd = this._durationService.duration(new Date(v.start), new Date(v.done));
                    let item = this.jobTime.get(k);
                    if (!item || item !== dd) {
                        this.jobTime.set(k, dd);
                    }
                    refresh = true;
                    break;
            }
        });

        if (!stillRunning) {
            this.deleteInterval();
            this._cd.markForCheck();
        }
        if (refresh) {
            this._cd.markForCheck();
        }
    }

    ngOnDestroy(): void {
        this.deleteInterval();
        this._store.dispatch(new SelectWorkflowNodeRunJob({ jobID: 0 }));
    }

    deleteInterval(): void {
        if (this.durationIntervalID) {
            clearInterval(this.durationIntervalID);
            this.durationIntervalID = 0;
        }
    }
}
