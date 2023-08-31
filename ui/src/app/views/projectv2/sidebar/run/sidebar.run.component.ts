import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input,
    OnDestroy,
    OnInit
} from "@angular/core";
import {AutoUnsubscribe} from "../../../../shared/decorator/autoUnsubscribe";
import {Project} from "../../../../model/project.model";
import {ActivatedRoute, Router} from "@angular/router";
import {RouterService} from "../../../../service/router/router.service";
import {Subscription} from "rxjs";
import {V2WorkflowRunService} from "../../../../service/workflowv2/workflow.service";
import {V2WorkflowRun, V2WorkflowRunJob} from "../../../../model/v2.workflow.run.model";
import {SidebarService} from "../../../../service/sidebar/sidebar.service";

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

    routeSub: Subscription;

    ngOnDestroy(): void {
    }

    constructor(private _cd: ChangeDetectorRef, private _routerService: RouterService, private _router: Router,
                private _workflowService: V2WorkflowRunService, private _sidebarService: SidebarService) {}

    ngOnInit(): void {
        let activatedRoute = this._routerService.getActivatedRoute(this._router.routerState.root);
        this.routeSub = activatedRoute.params.subscribe(p => {
            this.listRun(activatedRoute);
        });
        activatedRoute.queryParams.subscribe(q => {
            this.listRun(activatedRoute);
        });
    }

    listRun(activatedRoute: ActivatedRoute) {
        let vcs = activatedRoute.snapshot.params["vcsName"];
        let repo = activatedRoute.snapshot.params["repoName"];
        let workflow = activatedRoute.snapshot.params["workflowName"];
        let branch = activatedRoute.snapshot.queryParams["branch"];

        if (vcs === this.currentVCSName && repo === this.currentRepoName && workflow === this.currentWorkflowName && branch === this.currentBranch) {
            return;
        }
        this.currentVCSName = vcs;
        this.currentRepoName = repo;
        this.currentWorkflowName = workflow;
        this.currentBranch = branch;

        this._workflowService.listRun(this.project.key, this.currentVCSName, this.currentRepoName, this.currentWorkflowName, this.currentBranch).subscribe(runs => {
            this.runs = runs;
            this._cd.markForCheck();
        });
    }

    selectRun(r: V2WorkflowRun): void {
        this.selectedRun = r;
        this._sidebarService.selectRun(r);
        this._cd.markForCheck();
    }
}
