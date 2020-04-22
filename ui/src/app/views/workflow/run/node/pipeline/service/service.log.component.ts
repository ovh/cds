import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    NgZone,
    OnDestroy, OnInit,
    ViewChild
} from '@angular/core';
import { Select, Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { PipelineStatus, ServiceLog } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-service-log',
    templateUrl: './service.log.html',
    styleUrls: ['service.log.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowServiceLogComponent implements OnDestroy, OnInit {

    @Select(WorkflowState.getSelectedWorkflowNodeJobRun()) nodeJobRun$: Observable<WorkflowNodeJobRun>;
    nodeJobRunSubs: Subscription;


    @ViewChild('logsContent') logsElt: ElementRef;

    logsSplitted: Array<string> = [];

    serviceLogs: Array<ServiceLog>;

    worker: CDSWebWorker;
    workerSubscription: Subscription;

    currentRunJobID: number;
    currentRunJobStatus: string;

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
            if (!njr) {
                this.stopWorker();
                return
            }
            if (this.currentRunJobID && njr.id === this.currentRunJobID && this.currentRunJobStatus === njr.status) {
                return;
            }
            this.currentRunJobID = njr.id;
            this.currentRunJobStatus = njr.status;
            if (!this.worker && (!this.serviceLogs || this.serviceLogs.length === 0)) {
                this.initWorker();
            }
            this._cd.markForCheck();
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
                runJobId: this.currentRunJobID,
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
                        if (this.currentRunJobStatus === PipelineStatus.SUCCESS || this.currentRunJobStatus === PipelineStatus.FAIL ||
                            this.currentRunJobStatus === PipelineStatus.STOPPED) {
                            this.stopWorker();
                        }
                        this._cd.markForCheck();
                    });
                }
            });
        }
    }

    ngOnDestroy() {
        this.stopWorker();
    }

    stopWorker() {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
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
