import {
    ChangeDetectionStrategy, ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    NgZone,
    OnDestroy,
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
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-step-log',
    templateUrl: './step.log.html',
    styleUrls: ['step.log.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowStepLogComponent implements OnInit, OnDestroy {

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

    worker: CDSWebWorker;
    workerSubscription: Subscription;
    queryParamsSubscription: Subscription;
    loading = true;
    loadingMore = false;
    startExec: Date;
    doneExec: Date;
    duration: string;
    selectedLine: number;
    splittedLogs: { lineNumber: number, value: string }[] = [];
    splittedLogsToDisplay: { lineNumber: number, value: string }[] = [];
    limitFrom: number;
    limitTo: number;
    basicView = false;
    allLogsView = false;
    ansiViewSelected = true;
    htmlViewSelected = true;
    ansi_up = new AnsiUp();

    zone: NgZone;
    _showLog = false;
    _force = false;
    _stepStatus: StepStatus;
    pipelineBuildStatusEnum = PipelineStatus;
    MAX_PRETTY_LOGS_LINES = 3500;
    @ViewChild('logsContent', { static: false }) logsElt: ElementRef;

    constructor(
        private _durationService: DurationService,
        private _router: Router,
        private _route: ActivatedRoute,
        private _hostElement: ElementRef,
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) {
        this.ansi_up.escape_for_html = !this.htmlViewSelected;
        this.zone = new NgZone({ enableLongStackTrace: false });
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
                        if (!this.stepStatus ) {
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
                this.showLogs = true;
                this.selectedLine = parseInt(qps['line'], 10);
            } else {
                this.selectedLine = null;
            }
        });
    }

    ngOnDestroy(): void {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
        if (this.worker) {
            this.worker.stop();
            this.worker = null;
        }
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

        if (!this.worker) {
            this.worker = new CDSWebWorker('./assets/worker/web/workflow-log.js');
            this.worker.start({
                key: this._store.selectSnapshot(ProjectState.projectSnapshot).key,
                workflowName: this._store.selectSnapshot(WorkflowState.workflowSnapshot).name,
                number: (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num,
                nodeRunId: (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id,
                runJobId: this.currentNodeJobRunID,
                stepOrder: this.stepOrder
            });

            this.workerSubscription = this.worker.response().subscribe(msg => {
                if (msg) {
                    let build: BuildResult = JSON.parse(String.raw`${msg}`);
                    this.zone.run(() => {
                        this._cd.markForCheck();
                        if (build.step_logs) {
                            this.logs = build.step_logs;
                            this.parseLogs();
                        }
                        if (this.loading) {
                            this.loading = false;
                            this.focusToLine();
                        }
                        if (!PipelineStatus.isActive(this.stepStatus.status)) {
                            this.worker.stop();
                        }
                    });
                }
            });
        }
    }

    htmlView() {
        this.ansiViewSelected = this.ansiViewSelected;
        this.htmlViewSelected = !this.htmlViewSelected;
        this.basicView = false;
        this.splittedLogs = null;
        this.parseLogs();
        this._cd.markForCheck();
    }

    ansiView() {
        this.ansiViewSelected = !this.ansiViewSelected;
        this.htmlViewSelected = this.htmlViewSelected;
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
        if ( (!this.splittedLogs && !tmpLogs) || (this.splittedLogs && tmpLogs && this.splittedLogs.length === tmpLogs.length)) {
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
                        return { lineNumber: this.splittedLogs.length + i, value: this.ansi_up.ansi_to_html(log) };
                    }
                    return { lineNumber: this.splittedLogs.length  + i, value: log };
            }));
        }
        if (!this.allLogsView && this.splittedLogs.length > this.MAX_PRETTY_LOGS_LINES && !this._route.snapshot.fragment) {
            this.limitFrom = 30;
            this.limitTo = this.splittedLogs.length - 40;
            this.splittedLogsToDisplay.splice(this.limitFrom, this.limitTo - this.limitFrom);
        } else {
            this.splittedLogsToDisplay = this.splittedLogs;
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
        if (!this.showLogs && this.worker) {
            this.workerSubscription.unsubscribe();
            this.worker.stop();
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
        setTimeout(() => {
            this.limitFrom = null;
            if (this.splittedLogs.length > this.MAX_PRETTY_LOGS_LINES) {
                this.basicView = true;
            }
            this.splittedLogsToDisplay = this.splittedLogs;
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
        let fragment = 'L' + this.job.pipeline_stage_id + '-' + this.job.pipeline_action_id + '-' + this.stepOrder + '-' + lineNumber;
        this._router.navigate([
            'project',
            this._store.selectSnapshot(ProjectState.projectSnapshot).key,
            'workflow',
            this._store.selectSnapshot(WorkflowState.workflowSnapshot).name,
            'run',
            (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num,
            'node',
            (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id
        ], { queryParams: qps, fragment });
    }
}
