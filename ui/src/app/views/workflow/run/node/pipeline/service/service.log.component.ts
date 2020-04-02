import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    NgZone,
    OnDestroy, OnInit,
    ViewChild
} from '@angular/core';
import * as AU from 'ansi_up';
import { PipelineStatus, ServiceLog } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { Observable, Subscription } from 'rxjs';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Select, Store } from '@ngxs/store';

@Component({
    selector: 'app-workflow-service-log',
    templateUrl: './service.log.html',
    styleUrls: ['service.log.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowServiceLogComponent implements OnDestroy, OnInit {

    @Select(WorkflowState.getSelectedWorkflowNodeJobRun()) nodeJobRun$: Observable<WorkflowNodeJobRun>;
    nodeJobRun: WorkflowNodeJobRun;
    nodeJobRunSubs: Subscription;


    @ViewChild('logsContent', { static: false }) logsElt: ElementRef;

    logsSplitted: Array<string> = [];

    serviceLogs: Array<ServiceLog>;

    worker: CDSWebWorker;
    workerSubscription: Subscription;

    showLog = {};
    loading = true;
    zone: NgZone;

    ansi_up = new AU.default;

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    ngOnInit(): void {
        this.nodeJobRunSubs = this.nodeJobRun$.subscribe(njr => {
            this.stopWorker();
            if (njr) {
                this.nodeJobRun = njr;
                if (PipelineStatus.isDone(njr.status)) {
                    this.stopWorker();
                }
            }
            this.initWorker();
        });
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
                key: this._store.selectSnapshot(ProjectState.projectSnapshot).key,
                workflowName: this._store.selectSnapshot(WorkflowState.workflowSnapshot).name,
                number: (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num,
                nodeRunId: (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id,
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
