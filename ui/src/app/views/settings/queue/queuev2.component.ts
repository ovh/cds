import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from "@angular/core";
import { QueueService, V2WorkflowRunService } from "app/service/services.module";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { lastValueFrom } from 'rxjs';
import { V2WorkflowRunJob, V2WorkflowRunJobStatus } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { PathItem } from "app/shared/breadcrumb/breadcrumb.component";
import { ActivatedRoute, Router } from "@angular/router";
import { NzTableFilterList, NzTableQueryParams } from "ng-zorro-antd/table";
import { NzMessageService } from "ng-zorro-antd/message";
import { ErrorUtils } from "app/shared/error.utils";

@Component({
    selector: 'app-queue-v2',
    templateUrl: './queuev2.component.html',
    styleUrls: ['./queuev2.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class QueueV2Component implements OnDestroy {

    jobs: Array<V2WorkflowRunJob> = [];
    loading: boolean;
    path: Array<PathItem>;
    pageIndex: number = 1;
    statusFilters: Array<string> = [];
    totalCount = 0;
    params: NzTableQueryParams;

    statusFilterList: NzTableFilterList = [
        { text: V2WorkflowRunJobStatus.Waiting, value: V2WorkflowRunJobStatus.Waiting, byDefault: true },
        { text: V2WorkflowRunJobStatus.Scheduling, value: V2WorkflowRunJobStatus.Scheduling, byDefault: true },
        { text: V2WorkflowRunJobStatus.Building, value: V2WorkflowRunJobStatus.Building }
    ];

    constructor(
        private _cd: ChangeDetectorRef,
        private _queueService: QueueService,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _workflowService: V2WorkflowRunService,
        private _messageService: NzMessageService
    ) {
        this.path = [<PathItem>{
            text: 'Settings'
        }, <PathItem>{
            text: 'Current Jobs V2 queue'
        }];

        this._activatedRoute.queryParams.subscribe(q => {
            this.pageIndex = q['page'] ?? 1;
            this.statusFilters = q['status'] ?? [];
            this.loadQueue();
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT 

    onQueryParamsChange(params: NzTableQueryParams): void {
        this.params = params;
        this.saveSearchInQueryParams();
    }

    async loadQueue() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            let offset = (this.pageIndex - 1) * 30;
            const resp = await lastValueFrom(this._queueService.getV2Jobs(this.statusFilters, null, offset, 30));
            this.totalCount = parseInt(resp.headers.get('X-Total-Count'), 10);
            this.jobs = resp.body;
        } catch (e) {
            this._messageService.error(`Unable to list workflow run jobs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        this.loading = false;
        this._cd.markForCheck();
    }

    async stopJob(job: V2WorkflowRunJob) {
        try {
            await lastValueFrom(this._workflowService.stopJob(job.project_key, job.workflow_run_id, job.id));
            await this.loadQueue();
        } catch (e) {
            this._messageService.error(`Unable to stop workflow run job: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    pageIndexChange(index: number): void {
        this.pageIndex = index;
        this._cd.markForCheck();
        this.saveSearchInQueryParams();
    }

    saveSearchInQueryParams() {
        let queryParams = {  };
        if (this.pageIndex > 1) {
            queryParams['page'] = this.pageIndex;
        }
        if (this.params) {
            this.params.filter.forEach(f => { queryParams[f.key] = f.value });
        }
        this._router.navigate([], {
            relativeTo: this._activatedRoute,
            queryParams
        });
    }
}