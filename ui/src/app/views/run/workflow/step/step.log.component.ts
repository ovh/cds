import {Component, Input, OnInit, OnDestroy, NgZone} from '@angular/core';
import {Action} from '../../../../model/action.model';
import {CDSWorker} from '../../../../shared/worker/worker';
import {Subscription} from 'rxjs/Rx';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
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

    constructor(private _authStore: AuthentificationStore) { }

    ngOnInit(): void {
        this.zone = new NgZone({enableLongStackTrace: false});
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

            this.worker.response().subscribe( msg => {
                if (msg) {
                    let build: BuildResult = JSON.parse(msg);
                    this.zone.run(() => {
                        if (build.step_logs) {
                            this.logs = build.step_logs;
                        }
                    });
                    if (this.loading) {
                        this.loading = false;
                    }
                }
            });
        }
    }

    ngOnDestroy(): void {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
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
