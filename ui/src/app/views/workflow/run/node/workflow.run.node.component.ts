import { Component } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationExtras, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { RouterService } from 'app/service/router/router.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { AuthenticationState } from 'app/store/authentication.state';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { GetWorkflowNodeRun, GetWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-run-node',
    templateUrl: './node.html',
    styleUrls: ['./node.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeRunComponent {

    node: WNode;
    nodeRun: WorkflowNodeRun;
    subNodeRun: Subscription;

    // Context info
    project: Project;
    project$: Subscription;
    workflowName: string;

    storeSub: Subscription;

    duration: string;

    workflowRun: WorkflowRun;
    pipelineName = '';

    // History
    nodeRunsHistory = new Array<WorkflowNodeRun>();
    selectedTab: string;

    isAdmin: boolean;

    nbVuln = 0;
    deltaVul = 0;

    constructor(
        private _store: Store,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _routerService: RouterService,
        private _workflowRunService: WorkflowRunService,
        private _durationService: DurationService,
        private _titleService: Title
    ) {

        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.project$ = this._store.select(ProjectState)
            .pipe(filter((prj) => prj != null))
            .subscribe((projState: ProjectStateModel) => {
                this.project = projState.project;
            });

        this.isAdmin = this._store.selectSnapshot(AuthenticationState.user).admin;

        // Tab selection
        this._activatedRoute.queryParams.subscribe(q => {
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'pipeline';
            }
            this.pipelineName = q['name'] || '';
        });

        // Get workflow name
        this.workflowName = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root)['workflowName'];
        let historyChecked = false;
        this.storeSub = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            if (!s.workflow || this.workflowName !== s.workflow.name) {
                return;
            }
            this.workflowRun = s.workflowRun;
            this.nodeRun = cloneDeep(s.workflowNodeRun);
            if (this.workflowRun && this.workflowRun.workflow && this.nodeRun) {
                this.node = Workflow.getNodeByID(this.nodeRun.workflow_node_id, this.workflowRun.workflow);
            }

            if (this.nodeRun) {
                if (!historyChecked) {
                    historyChecked = true;
                    this._workflowRunService.nodeRunHistory(
                        this.project.key, this.workflowName,
                        this.nodeRun.num, this.nodeRun.workflow_node_id)
                        .pipe(first())
                        .subscribe(nrs => this.nodeRunsHistory = nrs);
                }
                this.initVulnerabilitySummary();
                if (this.nodeRun && !PipelineStatus.isActive(this.nodeRun.status)) {
                    this.duration = this._durationService.duration(new Date(this.nodeRun.start), new Date(this.nodeRun.done));
                }
                this.updateTitle();
            }
        });

        this._activatedRoute.params.subscribe(params => {
            this.nodeRun = null;
            let number = params['number'];
            let nodeRunId = params['nodeId'];

            if (this.project && this.project.key && this.workflowName && number && nodeRunId) {
                this._store.dispatch(new GetWorkflowRun({ projectKey: this.project.key, workflowName: this.workflowName, num: number }))
                    .subscribe(() => {
                        this._store.dispatch(
                            new GetWorkflowNodeRun({
                                projectKey: this.project.key,
                                workflowName: this.workflowName,
                                num: number,
                                nodeRunID: nodeRunId
                            }));
                    });

            }
        });
    }

    showTab(tab: string): void {
        let queryParams = Object.assign({}, this._activatedRoute.snapshot.queryParams, { tab })
        let navExtras: NavigationExtras = { queryParams };
        this._router.navigate(['project', this.project.key,
            'workflow', this.workflowName,
            'run', this.nodeRun.num,
            'node', this.nodeRun.id], navExtras);
    }

    updateTitle() {
        if (!this.workflowRun || !Array.isArray(this.workflowRun.tags)) {
            return;
        }
        let branch = this.workflowRun.tags.find((tag) => tag.tag === 'git.branch');
        if (branch) {
            this._titleService
                .setTitle(`Pipeline ${this.pipelineName} • #${this.workflowRun.num} [${branch.value}] • ${this.workflowName}`);
        }
    }

    initVulnerabilitySummary(): void {
        if (this.nodeRun && this.nodeRun.vulnerabilities_report && this.nodeRun.vulnerabilities_report.report) {
            if (this.nodeRun.vulnerabilities_report.report.summary) {
                Object.keys(this.nodeRun.vulnerabilities_report.report.summary).forEach(k => {
                    this.nbVuln += this.nodeRun.vulnerabilities_report.report.summary[k];
                });
            }
            let previousNb = 0;
            if (this.nodeRun.vulnerabilities_report.report.previous_run_summary) {
                Object.keys(this.nodeRun.vulnerabilities_report.report.previous_run_summary).forEach(k => {
                    previousNb += this.nodeRun.vulnerabilities_report.report.previous_run_summary[k];
                });
            } else if (this.nodeRun.vulnerabilities_report.report.default_branch_summary) {
                Object.keys(this.nodeRun.vulnerabilities_report.report.default_branch_summary).forEach(k => {
                    previousNb += this.nodeRun.vulnerabilities_report.report.default_branch_summary[k];
                });
            }
            this.deltaVul = this.nbVuln - previousNb;
        }
    }
}
