import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    NgZone,
    OnInit,
    ViewChild
} from '@angular/core';
import { Select, Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { PipelineStatus, ServiceLog } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
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
export class WorkflowServiceLogComponent implements OnInit {
    @Select(WorkflowState.getSelectedWorkflowNodeJobRun()) nodeJobRun$: Observable<WorkflowNodeJobRun>;
    nodeJobRunSubs: Subscription;

    @ViewChild('logsContent') logsElt: ElementRef;

    logsSplitted: Array<string> = [];

    serviceLogs: Array<ServiceLog>;

    pollingSubscription: Subscription;

    currentRunJobID: number;
    currentRunJobStatus: string;

    showLog = {};
    loading = true;
    zone: NgZone;

    ansi_up = new AU.default;

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _ngZone: NgZone,
        private _workflowService: WorkflowService
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    ngOnInit(): void {
        this.nodeJobRunSubs = this.nodeJobRun$.subscribe(njr => {
            if (!njr) {
                this.stopPolling();
                return
            }
            if (this.currentRunJobID && njr.id === this.currentRunJobID && this.currentRunJobStatus === njr.status) {
                return;
            }
            this.currentRunJobID = njr.id;
            this.currentRunJobStatus = njr.status;
            if (!this.pollingSubscription && (!this.serviceLogs || this.serviceLogs.length === 0)) {
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

        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let runNumber = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num;
        let nodeRunId = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id;
        let runJobId = this.currentRunJobID;

        let callback = (serviceLogs: Array<ServiceLog>) => {
            this.serviceLogs = serviceLogs.map((log, id) => {
                this.showLog[id] = this.showLog[id] || false;
                log.logsSplitted = this.getLogs(log).split('\n');
                return log;
            });
            if (this.loading) {
                this.loading = false;
            }
            this._cd.markForCheck();
        };

        this._workflowService.getServiceLog(projectKey, workflowName, runNumber, nodeRunId, runJobId).subscribe(callback);

        if (this.currentRunJobStatus === PipelineStatus.SUCCESS
            || this.currentRunJobStatus === PipelineStatus.FAIL
            || this.currentRunJobStatus === PipelineStatus.STOPPED) {
            return;
        }

        this.stopPolling();
        this._ngZone.runOutsideAngular(() => {
            this.pollingSubscription = Observable.interval(2000)
                .mergeMap(_ => this._workflowService.getServiceLog(projectKey, workflowName, runNumber, nodeRunId, runJobId))
                .subscribe(serviceLogs => {
                    this.zone.run(() => {
                        callback(serviceLogs);
                        if (this.currentRunJobStatus === PipelineStatus.SUCCESS
                            || this.currentRunJobStatus === PipelineStatus.FAIL
                            || this.currentRunJobStatus === PipelineStatus.STOPPED) {
                            this.stopPolling();
                        }
                    });
                });
        });
    }

    stopPolling() {
        if (this.pollingSubscription) {
            this.pollingSubscription.unsubscribe();
        }
    }

    copyRawLog(serviceLog) {
        this.logsElt.nativeElement.value = serviceLog.val;
        this.logsElt.nativeElement.select();
        document.execCommand('copy');
    }
}
