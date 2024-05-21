import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from "@angular/core";
import { QueueService, V2WorkflowRunService } from "app/service/services.module";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { ToastService } from "app/shared/toast/ToastService";
import { finalize } from "rxjs";
import { V2WorkflowRunJob, V2WorkflowRunJobStatus } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { PathItem } from "app/shared/breadcrumb/breadcrumb.component";
import { ActivatedRoute, Router } from "@angular/router";
import { ThisReceiver } from "@angular/compiler";

@Component({
    selector: 'app-queue-v2',
    templateUrl: './queuev2.component.html',
    styleUrls: ['./queuev2.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class QueueV2Component implements OnDestroy {

    jobs: V2WorkflowRunJob[];

    allStatus: string[] = [V2WorkflowRunJobStatus.Waiting, V2WorkflowRunJobStatus.Building, V2WorkflowRunJobStatus.Scheduling];

    statusFilter: string[];
    regionFilter: string[];

    loading: boolean;
    offset = 0;
    pageSize = 10;
    pageIndex = 1;

    path: Array<PathItem>;
    totalJobs = 0;

    constructor(private _cd: ChangeDetectorRef, private _toast: ToastService, private _queueService: QueueService, 
        private _routerActivated: ActivatedRoute, private _router: Router, private _workflowService: V2WorkflowRunService) {
        this.statusFilter = [];
        this.path = [<PathItem>{
            translate: 'Settings'
        }, <PathItem>{
            translate: 'Current Jobs V2 queue'
        }];

        this._routerActivated.queryParams.subscribe(q => {
            this.pageIndex = q['page'] ?? 1;
            this.statusFilter = q['status'] ?? [V2WorkflowRunJobStatus.Waiting, V2WorkflowRunJobStatus.Scheduling]
            this.loadQueue(this.pageIndex);
        });
    }
    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT 

    loadQueue(page: number): void {
        console.log(this.statusFilter);
        this.loading = true;
        this._cd.markForCheck();
        let offset = (page-1) * this.pageSize;
        this._queueService.getV2Jobs(this.statusFilter, this.regionFilter, offset, this.pageSize)
        .pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
        .subscribe(resp => {
            this.totalJobs = parseInt(resp.headers.get('X-Total-Count'), 10);
            this.jobs = resp.body; 
            this.pageIndex = page;
        })
    }

    changePage(newpage: any): void {
        this._router.navigate([], {
			relativeTo: this._routerActivated,
			queryParams: {'page': newpage},
			replaceUrl: true,
		});
    }

    updateFilters(): void {
        this._router.navigate([], {
			relativeTo: this._routerActivated,
			queryParams: {'page': 1, 'status': this.statusFilter},
			replaceUrl: true
		});
    }

    reloadPage(): void {
        this.loadQueue(this.pageIndex);
    }

    stopJob(job: V2WorkflowRunJob): void {
        this._workflowService.stopJob(job.project_key, job.workflow_run_id, job.id).subscribe(() => {
            if (this.jobs.length === 1 && this.pageIndex > 1) {
                this.pageIndex--;
            }
            this.reloadPage();
        });
    }
}