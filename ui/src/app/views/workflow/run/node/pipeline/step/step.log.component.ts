import {Component, Input, OnInit, NgZone, ViewChild, ElementRef, OnDestroy} from '@angular/core';
import {Router, ActivatedRoute} from '@angular/router';
import {Subscription} from 'rxjs';
import {Action} from '../../../../../../model/action.model';
import {Project} from '../../../../../../model/project.model';
import {Job, StepStatus} from '../../../../../../model/job.model';
import {BuildResult, Log, PipelineStatus} from '../../../../../../model/pipeline.model';
import {WorkflowNodeJobRun, WorkflowNodeRun} from '../../../../../../model/workflow.run.model';
import {CDSWorker} from '../../../../../../shared/worker/worker';
import {AutoUnsubscribe} from '../../../../../../shared/decorator/autoUnsubscribe';
import {AuthentificationStore} from '../../../../../../service/auth/authentification.store';
import {DurationService} from '../../../../../../shared/duration/duration.service';
import {environment} from '../../../../../../../environments/environment';

declare var ansi_up: any;

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

    worker: CDSWorker;
    workerSubscription: Subscription;
    queryParamsSubscription: Subscription;
    loading = true;
    startExec: Date;
    doneExec: Date;
    duration: string;
    selectedLine: number;

    zone: NgZone;
    _showLog = false;
    _force = false;
    _stepStatus: StepStatus;
    pipelineBuildStatusEnum = PipelineStatus;
    @ViewChild('logsContent') logsElt: ElementRef;

    constructor(
        private _authStore: AuthentificationStore,
        private _durationService: DurationService,
        private _router: Router,
        private _route: ActivatedRoute,
        private _hostElement: ElementRef
      ) { }

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
            this.worker = new CDSWorker('./assets/worker/web/workflow-log.js');
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
                    let build: BuildResult = JSON.parse(msg);
                    this.zone.run(() => {
                        if (build.step_logs) {
                            this.logs = build.step_logs;
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

    getLogs() {
        if (this.logs && this.logs.val) {
            return ansi_up.ansi_to_html(this.logs.val);
        }
        return '';
    }

    getLogsSplitted() {
      return this.getLogs().split('\n');
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
