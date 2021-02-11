import { HttpClient } from '@angular/common/http';
import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    NgZone,
    OnDestroy,
    OnInit,
    ViewChild
} from '@angular/core';
import { Select, Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { CDNLogLink, PipelineStatus, ServiceLog } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { FeatureNames } from 'app/service/feature/feature.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { FeatureState } from 'app/store/feature.state';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { interval, Subscription } from 'rxjs';
import { map, mergeMap } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-service-log',
    templateUrl: './service.log.html',
    styleUrls: ['service.log.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowServiceLogComponent implements OnInit, OnDestroy {
    @ViewChild('logsContent') logsElt: ElementRef;

    @Input() serviceName: string;

    logsSplitted: Array<string> = [];
    serviceLog: ServiceLog;

    pollingSubscription: Subscription;

    currentRunJobID: number;
    currentRunJobStatus: string;

    showLog: boolean;
    loading = true;
    zone: NgZone;

    ansi_up = new AU.default();

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _ngZone: NgZone,
        private _workflowService: WorkflowService,
        private _http: HttpClient
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        let njr = this._store.selectSnapshot(WorkflowState.getSelectedWorkflowNodeJobRun());
        if (!njr) {
            this.stopPolling();
            return
        }
        if (this.currentRunJobID && njr.id === this.currentRunJobID && this.currentRunJobStatus === njr.status) {
            return;
        }

        let invalidServiceName = !njr.job.action.requirements.find(r => r.type === 'service' && r.name === this.serviceName);
        if (invalidServiceName) {
            return;
        }

        this.currentRunJobID = njr.id;
        this.currentRunJobStatus = njr.status;
        if (!this.pollingSubscription) {
            this.initWorker();
        }
        this._cd.markForCheck();
    }

    getLogs(serviceLog: ServiceLog) {
        if (serviceLog && serviceLog.val) {
            return this.ansi_up.ansi_to_html(serviceLog.val);
        }
        return '';
    }

    async initWorker() {
        if (!this.serviceLog) {
            this.loading = true;
        }

        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let nodeRunId = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id;
        let runJobId = this.currentRunJobID;

        const featCDN = this._store.selectSnapshot(FeatureState.featureProject(FeatureNames.CDNJobLogs,
            JSON.stringify({ project_key: projectKey })))
        const cdnEnabled = featCDN && (!featCDN?.exists || featCDN.enabled);

        let logLink: CDNLogLink;
        if (cdnEnabled) {
            logLink = await this._workflowService.getServiceLink(projectKey, workflowName, nodeRunId, runJobId,
                this.serviceName).toPromise();
        }

        let callback = (serviceLog: ServiceLog) => {
            this.serviceLog = serviceLog;
            this.logsSplitted = this.getLogs(serviceLog).split('\n');
            if (this.loading) {
                this.loading = false;
            }
            this._cd.markForCheck();
        };

        if (!cdnEnabled) {
            const serviceLog = await this._workflowService.getServiceLog(projectKey, workflowName,
                nodeRunId, runJobId, this.serviceName).toPromise();
            callback(serviceLog);
        } else {
            const data = await this._workflowService.getLogDownload(logLink).toPromise();
            callback(<ServiceLog>{ val: data });
        }

        if (this.currentRunJobStatus === PipelineStatus.SUCCESS
            || this.currentRunJobStatus === PipelineStatus.FAIL
            || this.currentRunJobStatus === PipelineStatus.STOPPED) {
            return;
        }

        this.stopPolling();
        this._ngZone.runOutsideAngular(() => {
            this.pollingSubscription = interval(2000)
                .pipe(
                    mergeMap(() => {
                        if (!cdnEnabled) {
                            return this._workflowService.getServiceLog(projectKey, workflowName, nodeRunId,
                                runJobId, this.serviceName);
                        }
                        return this._workflowService.getLogDownload(logLink).pipe(map(data => <ServiceLog>{ val: data }));
                    })
                )
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
