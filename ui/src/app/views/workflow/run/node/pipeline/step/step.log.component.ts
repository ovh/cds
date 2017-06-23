import {Component, Input, OnInit,  NgZone} from '@angular/core';
import {Subscription} from 'rxjs/Rx';
import {Action} from '../../../../../../model/action.model';
import {Project} from '../../../../../../model/project.model';
import {BuildResult, Log} from '../../../../../../model/pipeline.model';
import {WorkflowNodeJobRun, WorkflowNodeRun} from '../../../../../../model/workflow.run.model';
import {CDSWorker} from '../../../../../../shared/worker/worker';
import {AutoUnsubscribe} from '../../../../../../shared/decorator/autoUnsubscribe';
import {AuthentificationStore} from '../../../../../../service/auth/authentification.store';
import {environment} from '../../../../../../../environments/environment';

declare var ansi_up: any;

@Component({
    selector: 'app-workflow-step-log',
    templateUrl: './step.log.html',
    styleUrls: ['step.log.scss']
})
@AutoUnsubscribe()
export class WorkflowStepLogComponent implements OnInit {

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
        if (data && !this.currentStatus) {
            this.initWorker();
        }
        this.currentStatus = data;
    }
    logs: Log;
    currentStatus: string;
    showLog = false;

    worker: CDSWorker;
    workerSubscription: Subscription;

    zone: NgZone;

    constructor(private _authStore: AuthentificationStore) { }

    ngOnInit(): void {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    initWorker(): void {
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

            this.workerSubscription = this.worker.response().subscribe( msg => {
                if (msg) {
                    let build: BuildResult = JSON.parse(msg);
                    this.zone.run(() => {
                        if (build.step_logs) {
                            this.logs = build.step_logs;
                        }
                    });
                }
            });
        }
    }

    toggleLogs() {
        this.showLog = ! this.showLog;
    }

    getLogs() {
        if (this.logs && this.logs.val) {
            return ansi_up.ansi_to_html(this.logs.val);
        }
        return '';
    }
}
