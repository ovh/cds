import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    NgZone,
    OnDestroy,
    ViewChild
} from '@angular/core';
import { Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { AuthenticationState } from 'app/store/authentication.state';
import { Subscription } from 'rxjs';
import { PipelineStatus, ServiceLog } from '../../../../../../model/pipeline.model';
import { Project } from '../../../../../../model/project.model';
import { WorkflowNodeJobRun, WorkflowNodeRun } from '../../../../../../model/workflow.run.model';
import { AutoUnsubscribe } from '../../../../../../shared/decorator/autoUnsubscribe';
import { CDSWebWorker } from '../../../../../../shared/worker/web.worker';

@Component({
    selector: 'app-workflow-service-log',
    templateUrl: './service.log.html',
    styleUrls: ['service.log.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowServiceLogComponent implements OnDestroy {

    @Input() project: Project;
    @Input() workflowName: string;
    @Input() nodeRun: WorkflowNodeRun;
    @Input('nodeJobRun')
    set nodeJobRun(data: WorkflowNodeJobRun) {
        this.stopWorker();
        if (data) {
            this._nodeJobRun = data;
            if (PipelineStatus.isDone(data.status)) {
                this.stopWorker();
            }
        }
        this.initWorker();
    }
    get nodeJobRun(): WorkflowNodeJobRun {
        return this._nodeJobRun;
    }

    @ViewChild('logsContent', { static: false }) logsElt: ElementRef;

    logsSplitted: Array<string> = [];

    serviceLogs: Array<ServiceLog>;

    worker: CDSWebWorker;
    workerSubscription: Subscription;

    showLog = {};
    loading = true;
    zone: NgZone;
    _nodeJobRun: WorkflowNodeJobRun;
    ansi_up = new AU.default;

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    getLogs(serviceLog: ServiceLog) {
        if (serviceLog && serviceLog.val) {
            return this.ansi_up.ansi_to_html(serviceLog.val);
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
                user: this._store.selectSnapshot(AuthenticationState.user),
                // TODO
                // session: this._authStore.getSessionToken(),
                api: '/cdsapi',
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
                        this._cd.markForCheck();
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
