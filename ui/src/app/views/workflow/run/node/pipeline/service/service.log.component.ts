import {Component, ElementRef, Input, NgZone, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {Subscription} from 'rxjs';
import {environment} from '../../../../../../../environments/environment';
import {ServiceLog} from '../../../../../../model/pipeline.model';
import {PipelineStatus} from '../../../../../../model/pipeline.model';
import {Project} from '../../../../../../model/project.model';
import {WorkflowNodeJobRun, WorkflowNodeRun} from '../../../../../../model/workflow.run.model';
import {AuthentificationStore} from '../../../../../../service/auth/authentification.store';
import {AutoUnsubscribe} from '../../../../../../shared/decorator/autoUnsubscribe';
import {CDSWebWorker} from '../../../../../../shared/worker/web.worker';

declare var ansi_up: any;

@Component({
    selector: 'app-workflow-service-log',
    templateUrl: './service.log.html',
    styleUrls: ['service.log.scss']
})
@AutoUnsubscribe()
export class WorkflowServiceLogComponent implements OnInit, OnDestroy {

    @Input() project: Project;
    @Input() workflowName: string;
    @Input() nodeRun: WorkflowNodeRun;
    @Input('nodeJobRun')
    set nodeJobRun(data: WorkflowNodeJobRun) {
        if (data) {
            this._nodeJobRun = data;
            if (data.status === PipelineStatus.SUCCESS || data.status === PipelineStatus.FAIL || data.status === PipelineStatus.STOPPED) {
                this.stopWorker();
            }
        }
    }
    get nodeJobRun(): WorkflowNodeJobRun {
        return this._nodeJobRun;
    }

    @ViewChild('logsContent') logsElt: ElementRef;

    logsSplitted: Array<string> = [];

    serviceLogs: Array<ServiceLog>;

    worker: CDSWebWorker;
    workerSubscription: Subscription;

    showLog = {};
    loading = true;
    zone: NgZone;
    _nodeJobRun: WorkflowNodeJobRun;

    constructor(private _authStore: AuthentificationStore) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnInit(): void {
        this.initWorker();
    }

    getLogs(serviceLog: ServiceLog) {
        if (serviceLog && serviceLog.val) {
            return ansi_up.ansi_to_html(serviceLog.val);
        }
        return '';
    }

    initWorker(): void {
        if (!this.serviceLogs) {
            this.loading = true;
        }

        if (!this.worker) {
            this.worker = new CDSWebWorker('./assets/worker/web/workflow-service-log.js');
            this.worker.start({
                user: this._authStore.getUser(),
                session: this._authStore.getSessionToken(),
                api: environment.apiURL,
                key: this.project.key,
                workflowName: this.workflowName,
                number: this.nodeRun.num,
                nodeRunId: this.nodeRun.id,
                runJobId: this.nodeJobRun.id,
            });

            this.workerSubscription = this.worker.response().subscribe(msg => {
                if (msg) {
                    let serviceLogs: Array<ServiceLog> = JSON.parse(msg);
                    this.zone.run(() => {
                        this.serviceLogs = serviceLogs.map((log, id) => {
                            this.showLog[id] = this.showLog[id] || false;
                            log.logsSplitted = this.getLogs(log).split('\n');
                            return log;
                        });
                        if (this.loading) {
                            this.loading = false;
                        }
                        if (this.nodeJobRun.status === PipelineStatus.SUCCESS || this.nodeJobRun.status === PipelineStatus.FAIL ||
                            this.nodeJobRun.status === PipelineStatus.STOPPED) {
                            this.stopWorker();
                        }
                    });
                }
            });
        }
    }

    ngOnDestroy() {
        this.stopWorker();
    }

    stopWorker() {
        if (this.worker) {
            this.worker.stop();
            this.worker = null;
        }
    }

    copyRawLog(serviceLog) {
        this.logsElt.nativeElement.value = serviceLog.val;
        this.logsElt.nativeElement.select();
        document.execCommand('copy');
    }
}
