import {
    ChangeDetectionStrategy, ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    NgZone,
    OnInit,
    ViewChild
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Select, Store } from '@ngxs/store';
import {
    default as AnsiUp
} from 'ansi_up';
import { Action } from 'app/model/action.model';
import { Job, StepStatus } from 'app/model/job.model';
import { BuildResult, Log, PipelineStatus } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-step-log',
    templateUrl: './step.log.html',
    styleUrls: ['step.log.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowStepLogComponent implements OnInit {

    // Static
    @Input() stepOrder: number;

    currentNodeJobRunID: number;
    job: Job;
    step: Action;
    stepStatus: StepStatus;

    @Select(WorkflowState.getSelectedWorkflowNodeJobRun()) nodeJobRun$: Observable<WorkflowNodeJobRun>;
    nodeJobRunSubs: Subscription;

    logs: Log;
    showLogs = false;

    pollingSubscription: Subscription;
    queryParamsSubscription: Subscription;
    loading = true;
    loadingMore = false;
    startExec: Date;
    doneExec: Date;
    duration: string;
    selectedLine: number;
    splittedLogs: { lineNumber: number, value: string }[];
    splittedLogsToDisplay: { lineNumber: number, value: string }[] = [];
    limitFrom: number;
    limitTo: number;
    basicView = false;
    allLogsView = false;
    ansiViewSelected = true;
    htmlViewSelected = true;
    ansi_up = new AnsiUp();

    _showLog = false;
    _force = false;
    _stepStatus: StepStatus;
    pipelineBuildStatusEnum = PipelineStatus;
    MAX_PRETTY_LOGS_LINES = 3500;
    @ViewChild('logsContent') logsElt: ElementRef;

    constructor(
        private _durationService: DurationService,
        private _router: Router,
        private _route: ActivatedRoute,
        private _hostElement: ElementRef,
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _ngZone: NgZone,
        private _workflowService: WorkflowService
    ) {
        this.ansi_up.escape_for_html = !this.htmlViewSelected;
    }

    ngOnInit(): void {
        this.nodeJobRunSubs = this.nodeJobRun$.subscribe(nrj => {
            if (!nrj) {
                return;
            }
            let refresh = false;
            if (this.currentNodeJobRunID !== nrj.id) {
                refresh = true;
                this.currentNodeJobRunID = nrj.id;
                this.job = nrj.job;
                if (this.job.action.actions.length >= this.stepOrder + 1) {
                    this.step = this.job.action.actions[this.stepOrder];
                }
                if (nrj.job.step_status && nrj.job.step_status.length >= this.stepOrder + 1) {
                    this.stepStatus = nrj.job.step_status[this.stepOrder];
                    this.computeDuration();
                }
                if (this.stepStatus) {
                    if (this.stepStatus.status === this.pipelineBuildStatusEnum.BUILDING ||
                        this.stepStatus.status === this.pipelineBuildStatusEnum.WAITING ||
                        (this.stepStatus.status === this.pipelineBuildStatusEnum.FAIL && !this.step.optional)) {
                        this.showLogs = true;
                        this.initWorker();
                    }
                }
            } else {
                // check if step status change
                if (nrj.job.step_status && nrj.job.step_status.length >= this.stepOrder + 1) {
                    let status = nrj.job.step_status[this.stepOrder].status;
                    if (!this.stepStatus || status !== this.stepStatus.status) {
                        if (!this.stepStatus) {
                            this.initWorker();
                            this.showLogs = true;
                        } else if (this.pipelineBuildStatusEnum.isActive(this.stepStatus.status) &&
                            this.pipelineBuildStatusEnum.isDone(status)) {
                            this.showLogs = false;
                        }
                        this.stepStatus = nrj.job.step_status[this.stepOrder];
                        this.computeDuration();
                        refresh = true;
                    }
                }
            }
            if (refresh) {
                this._cd.markForCheck();
            }
        });

        this.queryParamsSubscription = this._route.queryParams.subscribe((qps) => {
            this._cd.markForCheck();
            let activeStep = parseInt(qps['stageId'], 10) === this.job.pipeline_stage_id &&
                parseInt(qps['actionId'], 10) === this.job.pipeline_action_id && parseInt(qps['stepOrder'], 10) === this.stepOrder;
            if (activeStep) {
                this.selectedLine = parseInt(qps['line'], 10);
                if (!this.showLogs) {
                    this.toggleLogs();
                }
            } else {
                this.selectedLine = null;
            }
        });
    }

    copyRawLog() {
        this.logsElt.nativeElement.value = this.logs.val;
        this.logsElt.nativeElement.select();
        document.execCommand('copy');
    }

    initWorker(): void {
        if (!this.logs) {
            this.loading = true;
        }

        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let runNumber = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num;
        let nodeRunId = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id;
        let runJobId = this.currentNodeJobRunID;
        let stepOrder = this.stepOrder;

        let callback = (b: BuildResult) => {
            if (b.step_logs.id) {
                this.logs = b.step_logs;
                this.parseLogs();
            }
            if (this.loading) {
                this.loading = false;
                this.focusToLine();
            }
        };

        this._workflowService.getStepLog(projectKey, workflowName, runNumber, nodeRunId, runJobId, stepOrder).subscribe(callback);

        if (!PipelineStatus.isActive(this.stepStatus.status)) {
            return;
        }

        this.stopWorker();
        this._ngZone.runOutsideAngular(() => {
            this.pollingSubscription = Observable.interval(2000)
                .mergeMap(_ => this._workflowService.getStepLog(projectKey, workflowName, runNumber, nodeRunId, runJobId, stepOrder))
                .subscribe(build => {
                    this._ngZone.run(() => {
                        callback(build);
                        if (!PipelineStatus.isActive(build.status) || !PipelineStatus.isActive(this.stepStatus.status)) {
                            this.stopWorker();
                        }
                    });
                });
        });
    }

    stopWorker() {
        if (this.pollingSubscription) {
            this.pollingSubscription.unsubscribe();
        }
    }

    htmlView() {
        this.htmlViewSelected = !this.htmlViewSelected;
        this.basicView = false;
        this.splittedLogs = null;
        this.parseLogs();
        this._cd.markForCheck();
    }

    ansiView() {
        this.ansiViewSelected = !this.ansiViewSelected;
        this.basicView = false;
        this.splittedLogs = null;
        this.parseLogs();
        this._cd.markForCheck();
    }

    rawView() {
        this.htmlViewSelected = false;
        this.ansiViewSelected = false;
        this.basicView = true;
        this.splittedLogs = null;
        this.parseLogs();
        this._cd.markForCheck();
    }

    parseLogs() {
        let tmpLogs = this.getLogsSplitted();
        if ((!this.splittedLogs && !tmpLogs) || (this.splittedLogs && tmpLogs && this.splittedLogs.length === tmpLogs.length)) {
            return;
        }
        if (!this.splittedLogs || this.splittedLogs.length > tmpLogs.length) {
            this.splittedLogs = tmpLogs.map((log, i) => {
                if (this.ansiViewSelected) {
                    return { lineNumber: i + 1, value: this.ansi_up.ansi_to_html(log) };
                }
                return { lineNumber: i + 1, value: log };
            });
        } else {
            this.splittedLogs.push(...tmpLogs.slice(this.splittedLogs.length).map((log, i) => {
                if (this.ansiViewSelected) {
                    return { lineNumber: this.splittedLogs.length + i + 1, value: this.ansi_up.ansi_to_html(log) };
                }
                return { lineNumber: this.splittedLogs.length + i + 1, value: log };
            }));
        }

        this.splittedLogsToDisplay = cloneDeep(this.splittedLogs);
        if (!this.allLogsView && this.splittedLogsToDisplay.length > this.MAX_PRETTY_LOGS_LINES && !this._route.snapshot.fragment) {
            this.limitFrom = 30;
            this.limitTo = this.splittedLogs.length - 40;
            this.splittedLogsToDisplay.splice(this.limitFrom, this.limitTo - this.limitFrom);
        }
        this._cd.markForCheck();
    }

    focusToLine() {
        if (this._route.snapshot.fragment) {
            setTimeout(() => {
                const element = this._hostElement.nativeElement.querySelector('#' + this._route.snapshot.fragment);
                if (element) {
                    element.scrollIntoView(true);
                    this._force = true;
                    this._cd.markForCheck();
                }
            });
        }
    }

    computeDuration() {
        if (!this.stepStatus || PipelineStatus.neverRun(this.stepStatus.status)) {
            return;
        }

        if (this.stepStatus.start && this.stepStatus.start.indexOf('0001-01-01') !== -1) {
            return;
        }
        this.startExec = this.stepStatus.start ? new Date(this.stepStatus.start) : new Date();

        if (this.stepStatus.done && this.stepStatus.done.indexOf('0001-01-01') !== -1) {
            this.doneExec = new Date();
        } else if (this.stepStatus.done) {
            this.doneExec = new Date(this.stepStatus.done);
        }

        if (this.doneExec) {
            this.duration = '(' + this._durationService.duration(this.startExec, this.doneExec) + ')';
        }
    }

    toggleLogs() {
        this._force = true;
        if (!this.showLogs && (!this.stepStatus || PipelineStatus.neverRun(this.stepStatus.status))) {
            return;
        }
        this.showLogs = !this.showLogs;
        if (!this.showLogs) {
            this.stopWorker();
        } else {
            this.initWorker();
        }
    }

    getLogs(): string {
        if (this.logs && this.logs.val) {
            return this.logs.val;
        }
        return '';
    }

    getLogsSplitted(): string[] {
        let l = this.getLogs();
        if (l.endsWith('\n')) {
            l = l.substr(0, l.length - 1);
        }
        return l.split('\n');
    }

    showAllLogs() {
        this.loadingMore = true;
        this.allLogsView = true;
        this._cd.markForCheck();
        setTimeout(() => {
            this.limitFrom = null;
            if (this.splittedLogs.length > this.MAX_PRETTY_LOGS_LINES) {
                this.basicView = true;
            }
            this.splittedLogsToDisplay = cloneDeep(this.splittedLogs);
            this.loadingMore = false;
            this._cd.markForCheck();
        }, 0);
    }

    generateLink(lineNumber: number) {
        let qps = Object.assign({}, this._route.snapshot.queryParams, {
            stageId: this.job.pipeline_stage_id,
            actionId: this.job.pipeline_action_id,
            stepOrder: this.stepOrder,
            line: lineNumber
        });
        this._router.navigate([
            'project',
            this._store.selectSnapshot(ProjectState.projectSnapshot).key,
            'workflow',
            this._store.selectSnapshot(WorkflowState.workflowSnapshot).name,
            'run',
            (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num,
            'node',
            (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id
        ], { queryParams: qps });
    }
}
