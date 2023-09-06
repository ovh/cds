import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit} from "@angular/core";
import {AutoUnsubscribe} from "../../../../shared/decorator/autoUnsubscribe";
import {Project} from "../../../../model/project.model";
import {ActivatedRoute, Router} from "@angular/router";
import {RouterService} from "../../../../service/router/router.service";
import {from, interval, of, Subscription} from "rxjs";
import {V2WorkflowRunService} from "../../../../service/workflowv2/workflow.service";
import {V2WorkflowRun} from "../../../../model/v2.workflow.run.model";
import {SidebarService} from "../../../../service/sidebar/sidebar.service";
import {concatMap, delay, mergeMap, repeat, tap} from "rxjs/operators";

@Component({
    selector: 'app-projectv2-sidebar-run',
    templateUrl: './sidebar.run.html',
    styleUrls: ['./sidebar.run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2SidebarRunComponent implements OnInit, OnDestroy {

    @Input() project: Project;

    currentVCSName: string;
    currentRepoName: string;
    currentWorkflowName: string;
    currentBranch: string;

    runs: Array<V2WorkflowRun>;
    selectedRun: V2WorkflowRun;
    selectedRunNumber: number;

    routeSub: Subscription;
    pollSub: Subscription;

    ngOnDestroy(): void {
    }

    constructor(private _cd: ChangeDetectorRef, private _routerService: RouterService, private _router: Router,
                private _workflowService: V2WorkflowRunService, private _sidebarService: SidebarService) {
    }

    ngOnInit(): void {
        let activatedRoute = this._routerService.getActivatedRoute(this._router.routerState.root);
        this.routeSub = activatedRoute.params.subscribe(p => {
            this.listRun(activatedRoute);
        });
        activatedRoute.queryParams.subscribe(q => {
            if (q['run']) {
                this.selectedRunNumber = q['run'];
            }
            this.listRun(activatedRoute);
        });
    }

    listRun(activatedRoute: ActivatedRoute) {
        let vcs = activatedRoute.snapshot.params["vcsName"];
        let repo = activatedRoute.snapshot.params["repoName"];
        let workflow = activatedRoute.snapshot.params["workflowName"];
        let branch = activatedRoute.snapshot.queryParams["branch"];

        if (vcs === this.currentVCSName && repo === this.currentRepoName && workflow === this.currentWorkflowName && branch === this.currentBranch) {
            if (this.selectedRunNumber && this.runs && this.selectedRunNumber !== this.selectedRun?.run_number) {
                this.runs.forEach(r => {
                    if (r.run_number === this.selectedRunNumber) {
                        this.selectRun(r);
                    }
                });
            }
            return;
        }
        this.currentVCSName = vcs;
        this.currentRepoName = repo;
        this.currentWorkflowName = workflow;
        this.currentBranch = branch;
        if (this.pollSub) {
            this.pollSub.unsubscribe();
        }
        this.pollSub = interval(5000)
            .pipe(concatMap(_ => from(this.loadRun())))
            .subscribe();
    }

    async loadRun() {
        this.runs = await this._workflowService.listRun(this.project.key, this.currentVCSName, this.currentRepoName, this.currentWorkflowName, this.currentBranch).toPromise();
        if (this.selectedRunNumber && this.runs) {
            this.runs.forEach(r => {
                if (r.run_number === this.selectedRunNumber && (this.selectedRunNumber !== this.selectedRun?.run_number || this.selectedRun?.status !== r.status)) {
                    this.selectRun(r);
                }
            });
        }
        this._cd.markForCheck();
    }

    selectRun(r: V2WorkflowRun): void {
        this.selectedRun = r;
        this.selectedRunNumber = r.run_number;
        this._sidebarService.selectRun(r);
        this._cd.markForCheck();
    }
}
