import {Component, Input, OnInit, OnDestroy, NgZone, ElementRef, ViewChild} from '@angular/core';
import {Action} from '../../../../model/action.model';
import {CDSWorker} from '../../../../shared/worker/worker';
import {Subscription} from 'rxjs/Rx';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {DurationService} from '../../../../shared/duration/duration.service';
import {environment} from '../../../../../environments/environment';
import {Project} from '../../../../model/project.model';
import {Application} from '../../../../model/application.model';
import {Pipeline, PipelineBuild, Log, BuildResult, PipelineStatus} from '../../../../model/pipeline.model';

declare var ansi_up: any;

@Component({
    selector: 'app-step-log',
    templateUrl: './step.log.html',
    styleUrls: ['step.log.scss']
})
export class StepLogComponent implements OnInit, OnDestroy {

    // Static
    @Input() step: Action;
    @Input() stepOrder: number;
    @Input() jobID: number;
    @Input() project: Project;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() pipelineBuild: PipelineBuild;
    @Input() previousBuild: PipelineBuild;
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

    zone: NgZone;
    _showLog = false;
    pipelineBuildStatusEnum = PipelineStatus;
    loading = true;
    startExec: Date;
    doneExec: Date;
    duration: string;
    intervalListener: any;

    @ViewChild('logsContent') logsElt: ElementRef;

    constructor(private _authStore: AuthentificationStore, private _durationService: DurationService) { }

    ngOnInit(): void {
        this.zone = new NgZone({enableLongStackTrace: false});
        if (this.currentStatus === this.pipelineBuildStatusEnum.BUILDING ||
            (this.currentStatus === this.pipelineBuildStatusEnum.FAIL && !this.step.optional)) {
          this.showLog = true;
        }
    }

    initWorker(): void {
        if (!this.logs) {
            this.loading = true;
        }
        if (!this.worker) {
            this.worker = new CDSWorker('./assets/worker/web/log.js');
            this.worker.start({
                user: this._authStore.getUser(),
                session: this._authStore.getSessionToken(),
                api: environment.apiURL,
                key: this.project.key,
                appName: this.application.name,
                pipName: this.pipeline.name,
                envName: this.pipelineBuild.environment.name,
                buildNumber: this.pipelineBuild.build_number,
                jobID: this.jobID,
                stepOrder: this.stepOrder
            });

            this.workerSubscription = this.worker.response().subscribe( msg => {
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

    copyRawLog() {
      this.logsElt.nativeElement.value = this.getLogs();
      this.logsElt.nativeElement.select();
      document.execCommand('copy');
    }

    ngOnDestroy(): void {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
        clearInterval(this.intervalListener);
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
