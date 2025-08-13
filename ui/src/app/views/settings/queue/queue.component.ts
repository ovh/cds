import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { EventType } from 'app/model/event.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { AuthSummary } from 'app/model/user.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { QueueService } from 'app/service/queue/queue.service';
import { V2WorkflowRunService } from 'app/service/services.module';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ErrorUtils } from 'app/shared/error.utils';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import { EventState } from 'app/store/event.state';
import moment from 'moment';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzTableQueryParams, NzTableFilterList } from 'ng-zorro-antd/table';
import { lastValueFrom, Subscription } from 'rxjs';
import { V2WorkflowRunJob, V2WorkflowRunJobStatus } from '../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model';
import { EventV2State } from 'app/store/event-v2.state';

@Component({
    selector: 'app-queue',
    templateUrl: './queue.component.html',
    styleUrls: ['./queue.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class QueueComponent implements OnDestroy {
    eventV1Subscription: Subscription;
    eventV2Subscription: Subscription;
    currentAuthSummary: AuthSummary;

    loading = {
        v1: true,
        v2: true
    };

    path: Array<PathItem>;

    jobsV1: Array<WorkflowNodeJobRun> = [];
    jobsV2: Array<V2WorkflowRunJob> = [];
    jobsV2totalCount = 0;
    pageIndexV1: number = 1;
    pageIndexV2: number = 1;
    statusFiltersV1: Array<string> = [];
    statusFiltersV2: Array<string> = [];
    paramsV1: NzTableQueryParams;
    paramsV2: NzTableQueryParams;
    statusFilterListV1: NzTableFilterList = [];
    statusFilterListV2: NzTableFilterList = [];

    constructor(
        private _activatedRoute: ActivatedRoute,
        private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _queueService: QueueService,
        private _router: Router,
        private _store: Store,
        private _toast: ToastService,
        private _wfRunService: WorkflowRunService,
        private _workflowService: V2WorkflowRunService
    ) {
        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            text: 'Current CDS jobs queue'
        }];

        const initialStatusV1 = this._activatedRoute.snapshot.queryParamMap.getAll("statusV1");
        this.statusFilterListV1 = [
            { text: PipelineStatus.WAITING, value: PipelineStatus.WAITING, byDefault: true },
            { text: PipelineStatus.BUILDING, value: PipelineStatus.BUILDING }
        ].map(f => ({
            ...f,
            byDefault: initialStatusV1.length > 0 ? initialStatusV1.indexOf(f.value) != -1 : f.byDefault
        }));

        const initialStatusV2 = this._activatedRoute.snapshot.queryParamMap.getAll("statusV2");
        this.statusFilterListV2 = [
            { text: V2WorkflowRunJobStatus.Waiting, value: V2WorkflowRunJobStatus.Waiting, byDefault: true },
            { text: V2WorkflowRunJobStatus.Scheduling, value: V2WorkflowRunJobStatus.Scheduling, byDefault: true },
            { text: V2WorkflowRunJobStatus.Building, value: V2WorkflowRunJobStatus.Building }
        ].map(f => ({
            ...f,
            byDefault: initialStatusV2.length > 0 ? initialStatusV2.indexOf(f.value) != -1 : f.byDefault
        }));

        this.eventV1Subscription = this._store.select(EventState.last).subscribe(e => {
            if (!e || e.type_event !== EventType.RUN_WORKFLOW_JOB) {
                return;
            }
            let jobID = e.payload['id'];
            if (e.status === PipelineStatus.WAITING || e.status === PipelineStatus.BUILDING) {
                try {
                    this._queueService.getJobInfos(jobID).subscribe(job => { this.appendOrUpdateJobV1(job); });
                } catch (e) {
                    this.removeJobV1(jobID);
                }
            } else {
                this.removeJobV1(jobID);
            }
            this._cd.markForCheck();
        });

        this.eventV2Subscription = this._store.select(EventV2State.last).subscribe(e => {
            if (!e || e.type.indexOf('RunJob') === -1) {
                return;
            }
            let jobID = e.payload['id'];
            if (e.status === V2WorkflowRunJobStatus.Waiting || e.status === V2WorkflowRunJobStatus.Scheduling || e.status === V2WorkflowRunJobStatus.Building) {
                this.appendOrUpdateJobV2(e.payload as V2WorkflowRunJob);
            } else {
                this.removeJobV2(jobID);
            }
            this._cd.markForCheck();
        });

        this._activatedRoute.queryParamMap.subscribe(q => {
            this.pageIndexV1 = q.get('pageV1') ? parseInt(q.get('pageV1'), 10) : 1;
            this.statusFiltersV1 = q.getAll('statusV1') ?? [];
            this.pageIndexV2 = q.get('pageV2') ? parseInt(q.get('pageV2'), 10) : 1;
            this.statusFiltersV2 = q.getAll('statusV2') ?? [];
            this.loadJobsV1();
            this.loadJobsV2();
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    async loadJobsV1() {
        this.loading.v1 = true;
        this._cd.markForCheck();

        try {
            const resp = await lastValueFrom(this._queueService.getWorkflows(this.statusFiltersV1));
            this.jobsV1 = resp.map(this.mapJobV1);
        } catch (e) {
            this._messageService.error(`Unable to list workflow run jobs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        this.loading.v1 = false;
        this._cd.markForCheck();
    }

    async loadJobsV2() {
        this.loading.v2 = true;
        this._cd.markForCheck();

        try {
            let offset = (this.pageIndexV2 - 1) * 100;
            const resp = await lastValueFrom(this._queueService.getV2Jobs(this.statusFiltersV2, null, offset, 100));
            this.jobsV2totalCount = parseInt(resp.headers.get('X-Total-Count'), 10);
            this.jobsV2 = resp.body;
        } catch (e) {
            this._messageService.error(`Unable to list workflow run jobs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        this.loading.v2 = false;
        this._cd.markForCheck();
    }

    async stopJobV1(job: WorkflowNodeJobRun) {
        job.updating = true;
        this.appendOrUpdateJobV1(job);
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._wfRunService.stopNodeRun(job.mParameters['cds.project'], job.mParameters['cds.workflow'],
                parseInt(job.mParameters['cds.run.number'], 10), parseInt(job.mParameters['cds.node.id'], 10)
            ));
            this._toast.success('', 'Job stopped');
        } catch (e) {
            this._messageService.error(`Unable to stop workflow run job: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    async stopJobV2(job: V2WorkflowRunJob) {
        job.updating = true;
        this.appendOrUpdateJobV2(job);
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._workflowService.stopJob(job.project_key, job.workflow_run_id, job.id));
            this._toast.success('', 'Job stopped');
        } catch (e) {
            this._messageService.error(`Unable to stop workflow run job: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    pageIndexV1Change(index: number): void {
        this.pageIndexV1 = index;
        this._cd.markForCheck();
        this.saveSearchInQueryParams();
    }

    pageIndexV2Change(index: number): void {
        this.pageIndexV2 = index;
        this._cd.markForCheck();
        this.saveSearchInQueryParams();
    }

    onQueryParamsV1Change(params: NzTableQueryParams): void {
        this.paramsV1 = params;
        this.saveSearchInQueryParams();
    }

    onQueryParamsV2Change(params: NzTableQueryParams): void {
        this.paramsV2 = params;
        this.saveSearchInQueryParams();
    }

    saveSearchInQueryParams() {
        let queryParams = {};
        if (this.pageIndexV1 > 1) {
            queryParams['pageV1'] = this.pageIndexV1;
        }
        if (this.pageIndexV2 > 1) {
            queryParams['pageV2'] = this.pageIndexV2;
        }
        if (this.paramsV1) {
            this.paramsV1.filter.forEach(f => { queryParams[f.key] = f.value });
        }
        if (this.paramsV2) {
            this.paramsV2.filter.forEach(f => { queryParams[f.key] = f.value });
        }
        this._router.navigate([], {
            relativeTo: this._activatedRoute,
            queryParams
        });
    }

    appendOrUpdateJobV1(job: WorkflowNodeJobRun) {
        this.jobsV1 = this.jobsV1.filter(j => j.id !== job.id).sort(this.sortJobsV1).concat(this.mapJobV1(job));
    }

    appendOrUpdateJobV2(job: V2WorkflowRunJob) {
        const previousLength = this.jobsV2.length;
        this.jobsV2 = this.jobsV2.filter(j => j.id !== job.id).concat(job).sort(this.sortJobsV2).slice(0, 100);
        if (this.jobsV2.length > previousLength) {
            this.jobsV2totalCount++;
        }
    }

    removeJobV1(jobID: number) {
        this.jobsV1 = this.jobsV1.filter(j => j.id !== jobID);
    }

    removeJobV2(jobID: string) {
        this.jobsV2 = this.jobsV2.filter(j => j.id !== jobID);
        this.jobsV2totalCount--;
    }

    mapJobV1(job: WorkflowNodeJobRun): WorkflowNodeJobRun {
        let bookedBySummary = '';
        if (job.status === PipelineStatus.BUILDING) {
            bookedBySummary = job.job.worker_name;
        } else if (job?.bookedby?.name) {
            bookedBySummary = job.bookedby.name;
        }
        let requirementsSummary = '';
        if (job.job && job.job.action && job.job.action.requirements) {
            requirementsSummary = job.job.action.requirements.map(req => `${req.type}=${req.value}`).join(', ');
        }
        return {
            ...job,
            bookedBySummary,
            requirementsSummary,
            mParameters: job.parameters.reduce((params, param) => {
                params[param.name] = param.value;
                return params;
            }, {})
        };
    }

    sortJobsV1(a: WorkflowNodeJobRun, b: WorkflowNodeJobRun) { return moment(a.queued).isBefore(moment(b.queued)) ? -1 : 1; }

    sortJobsV2(a: V2WorkflowRunJob, b: V2WorkflowRunJob) { return moment(a.queued).isBefore(moment(b.queued)) ? -1 : 1; }

    filterJobsV1(statuses: string[], job: WorkflowNodeJobRun) { return statuses.find(s => s === job.status); }
}
