import {Component, Input, OnInit, OnDestroy, NgZone, ViewChild, ElementRef} from '@angular/core';
import {Subscription} from 'rxjs/Rx';
import {Action} from '../../../../../../model/action.model';
import {Project} from '../../../../../../model/project.model';
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
    @Input() project: Project;
    @Input() workflowName: string;
    @Input() nodeRun: WorkflowNodeRun;
    @Input() nodeJobRun: WorkflowNodeJobRun;

    // Dynamic
    @Input('stepStatus')
    set stepStatus (data: string) {
        if (data && !this.currentStatus && this.showLog) {
            this.initWorker();
        }
        this.currentStatus = data;
    }
    logs: Log;
    currentStatus: string;
    set showLog(data: boolean) {
        this._showLog = data;

        if (data) {
            this.initWorker();
        } else {
            if (this.worker) {
                this.worker.stop();
            }
        }
    }
    get showLog() {
      return this._showLog;
    }

    worker: CDSWorker;
    workerSubscription: Subscription;
    loading = true;
    startExec: Date;
    doneExec: Date;
    duration: string;
    intervalListener: any;

    zone: NgZone;
    _showLog = false;
    pipelineBuildStatusEnum = PipelineStatus;
    @ViewChild('logsContent') logsElt: ElementRef;

    constructor(private _authStore: AuthentificationStore, private _durationService: DurationService) { }

    ngOnInit(): void {
        this.zone = new NgZone({enableLongStackTrace: false});
        if (this.currentStatus === this.pipelineBuildStatusEnum.BUILDING ||
            (this.currentStatus === this.pipelineBuildStatusEnum.FAIL && !this.step.optional)) {
          this.showLog = true;
        }
    }

    ngOnDestroy() {
        clearInterval(this.intervalListener);
    }

    copyRawLog() {
      this.logsElt.nativeElement.value = this.getLogs();
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
                            this.computeDuration();
                            this.loading = false;
                        }
                    });
                }
            });
        }
    }

    computeDuration() {
        this.startExec = new Date(this.logs.start.seconds * 1000);
        this.doneExec = this.logs.done && this.logs.done.seconds ? new Date(this.logs.done.seconds * 1000) : null;
        if (!this.duration) {
            this.duration = '(' + this._durationService.duration(this.startExec, this.doneExec || new Date()) + ')';
        }
        this.intervalListener = setInterval(() => {
            this.duration = '(' + this._durationService.duration(this.startExec, this.doneExec || new Date()) + ')';
            if (this.currentStatus !== PipelineStatus.BUILDING && this.currentStatus !== PipelineStatus.WAITING) {
                clearInterval(this.intervalListener);
            }
        }, 5000);
    }

    toggleLogs() {
        this.showLog = !this.showLog;
    }

    getLogs() {
        if (this.logs && this.logs.val) {
            return ansi_up.ansi_to_html(this.logs.val);
        }
        return '';
    }
}
