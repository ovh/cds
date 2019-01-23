import { Component, ElementRef, Input, NgZone, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import * as AU from 'ansi_up';
import { cloneDeep } from 'lodash';
import { Subscription } from 'rxjs';
import { environment } from '../../../../../../../environments/environment';
import { Action } from '../../../../../../model/action.model';
import { Job, StepStatus } from '../../../../../../model/job.model';
import { BuildResult, Log, PipelineStatus } from '../../../../../../model/pipeline.model';
import { Project } from '../../../../../../model/project.model';
import { WorkflowNodeJobRun, WorkflowNodeRun } from '../../../../../../model/workflow.run.model';
import { AuthentificationStore } from '../../../../../../service/auth/authentification.store';
import { AutoUnsubscribe } from '../../../../../../shared/decorator/autoUnsubscribe';
import { DurationService } from '../../../../../../shared/duration/duration.service';
import { CDSWebWorker } from '../../../../../../shared/worker/web.worker';

@Component({
    selector: 'app-workflow-step-log',
    templateUrl: './step.log.html',
    styleUrls: ['step.log.scss']
})
@AutoUnsubscribe()
export class WorkflowStepLogComponent implements OnInit, OnDestroy {

    // Static
    @Input() step: Action;
    @Input() stepOrder: number;
    @Input() job: Job;
    @Input() project: Project;
    @Input() workflowName: string;
    @Input() nodeRun: WorkflowNodeRun;
    @Input() nodeJobRun: WorkflowNodeJobRun;

    // Dynamic
    @Input('stepStatus')
    set stepStatus (data: StepStatus) {
        if (data && data.status && !this.currentStatus && this.showLog) {
            this.initWorker();
        }
        if (data) {
            this.currentStatus = data.status;
            if (!this._force  && PipelineStatus.isActive(data.status)) {
                this.showLog = true;
            }
        }
        this._stepStatus = data;
        this.computeDuration();
    }
    get stepStatus() {
        return this._stepStatus;
    }
    logs: Log;
    currentStatus: string;
    set showLog(data: boolean) {
        let neverRun = PipelineStatus.neverRun(this.currentStatus);
        if (data && !neverRun) {
            this.initWorker();
        } else {
            if (this.worker) {
                this.worker.stop();
                this.worker = null;
            }
        }
        this._showLog = data;
    }
    get showLog() {
      return this._showLog;
    }

    worker: CDSWebWorker;
    workerSubscription: Subscription;
    queryParamsSubscription: Subscription;
    loading = true;
    loadingMore = false;
    startExec: Date;
    doneExec: Date;
    duration: string;
    selectedLine: number;
    splittedLogs: {lineNumber: number, value: string}[] = [];
    splittedLogsToDisplay: {lineNumber: number, value: string}[] = [];
    limitFrom: number;
    limitTo: number;
    basicView = false;
    allLogsView = false;
    ansiViewSelected = true;
    htmlViewSelected = true;
    ansi_up = new AU.default;

    zone: NgZone;
    _showLog = false;
    _force = false;
    _stepStatus: StepStatus;
    pipelineBuildStatusEnum = PipelineStatus;
    MAX_PRETTY_LOGS_LINES = 3500;
    @ViewChild('logsContent') logsElt: ElementRef;

    constructor(
        private _authStore: AuthentificationStore,
        private _durationService: DurationService,
        private _router: Router,
        private _route: ActivatedRoute,
        private _hostElement: ElementRef
      ) {
          this.ansi_up.escape_for_html = !this.htmlViewSelected;
      }

    ngOnInit(): void {
        let nodeRunDone = this.nodeRun.status !== this.pipelineBuildStatusEnum.BUILDING &&
          this.nodeRun.status !== this.pipelineBuildStatusEnum.WAITING;
        let isLastStep = this.stepOrder === this.job.action.actions.length - 1;

        this.zone = new NgZone({enableLongStackTrace: false});
        if (this.currentStatus === this.pipelineBuildStatusEnum.BUILDING || this.currentStatus === this.pipelineBuildStatusEnum.WAITING ||
            (this.currentStatus === this.pipelineBuildStatusEnum.FAIL && !this.step.optional) ||
            (nodeRunDone && isLastStep && !PipelineStatus.neverRun(this.currentStatus))) {
          this.showLog = true;
        }

        this.queryParamsSubscription = this._route.queryParams.subscribe((qps) => {
          let activeStep = parseInt(qps['stageId'], 10) === this.job.pipeline_stage_id &&
            parseInt(qps['actionId'], 10) === this.job.pipeline_action_id && parseInt(qps['stepOrder'], 10) === this.stepOrder;

          if (activeStep) {
            this.showLog = true;
            this.selectedLine = parseInt(qps['line'], 10);
          } else {
            this.selectedLine = null;
          }
        });
    }

    ngOnDestroy(): void {
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
                user: this._authStore.getUser(),
                session: this._authStore.getSessionToken(),
                api: environment.apiURL,
                key: this.project.key,
                workflowName: this.workflowName,
                number: this.nodeRun.num,
                nodeRunId: this.nodeRun.id,
                runJobId: this.nodeJobRun.id,
                stepOrder: this.stepOrder
            });

            this.workerSubscription = this.worker.response().subscribe(msg => {
                if (msg) {
                    let build: BuildResult = JSON.parse(String.raw`${msg}`);
                    this.zone.run(() => {
                        if (build.step_logs) {
                            this.logs = build.step_logs;
                            this.parseLogs();
                        }
                        if (this.loading) {
                            this.loading = false;
                            this.focusToLine();
                        }
                    });
                }
            });
        }
    }

    htmlView() {
        this.htmlViewSelected = !this.htmlViewSelected;
        this.basicView = false;
        this.ansi_up.escape_for_html = !this.htmlViewSelected;
        this.parseLogs();
    }

    ansiView() {
        this.ansiViewSelected = !this.ansiViewSelected;
        this.basicView = false;
        this.ansi_up.escape_for_html = !this.htmlViewSelected;
        this.parseLogs();
    }

    parseLogs() {
        this.splittedLogs = this.getLogsSplitted()
            .map((log, i) => {
                if (this.ansiViewSelected) {
                    return {lineNumber: i + 1, value: this.ansi_up.ansi_to_html(log)};
                }
                return {lineNumber: i + 1, value: log};
            });
        this.splittedLogsToDisplay = cloneDeep(this.splittedLogs);

        if (!this.allLogsView && this.splittedLogs.length > 1000 && !this._route.snapshot.fragment) {
            this.limitFrom = 30;
            this.limitTo = this.splittedLogs.length - 40;
            this.splittedLogsToDisplay.splice(this.limitFrom, this.limitTo - this.limitFrom);
        } else {
            this.splittedLogsToDisplay = this.splittedLogs;
        }
    }

    focusToLine() {
      if (this._route.snapshot.fragment) {
        setTimeout(() => {
          const element = this._hostElement.nativeElement.querySelector('#' + this._route.snapshot.fragment);
          if (element) {
            element.scrollIntoView(true);
            this._force = true;
          }
        });
      }
    }

    computeDuration() {
        if (!this.stepStatus || PipelineStatus.neverRun(this.currentStatus)) {
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
        if (!this.showLog && PipelineStatus.neverRun(this.currentStatus)) {
            return;
        }
        this.showLog = !this.showLog;
    }

    getLogs(): string {
        if (this.logs && this.logs.val) {
            return this.logs.val;
        }
        return '';
    }

    getLogsSplitted(): string[] {
      return this.getLogs().split('\n');
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
        this.project.key,
        'workflow',
        this.workflowName,
        'run',
        this.nodeRun.num,
        'node',
        this.nodeRun.id
      ], {queryParams: qps, fragment});
    }
}
