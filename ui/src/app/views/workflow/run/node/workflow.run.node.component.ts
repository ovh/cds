import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationExtras, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { RouterService } from 'app/service/router/router.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { AuthenticationState } from 'app/store/authentication.state';
import { ProjectState } from 'app/store/project.state';
import { GetWorkflowNodeRun, GetWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-run-node',
    templateUrl: './node.html',
    styleUrls: ['./node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeRunComponent {
    nodeRun: WorkflowNodeRun;

    // Context info
    project: Project;
    project$: Subscription;
    workflowName: string;

    commitsLength: number = 0;
    artifactLength: number = 0;
    staticFilesLength: number = 0;
    historyLength: number = 0;

    storeSub: Subscription;

    pipelineName = '';
    pipelineStatus = PipelineStatus;

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
        private _titleService: Title,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.isAdmin = this._store.selectSnapshot(AuthenticationState.user).ring === 'ADMIN';

        // Tab selection
        this._activatedRoute.queryParams.subscribe(q => {
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'pipeline';
            }
            this.pipelineName = q['name'] || '';
            this._cd.markForCheck();
        });

        // Get workflow name
        this.workflowName = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root)['workflowName'];


        this.storeSub = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            if (!s.workflow || this.workflowName !== s.workflow.name) {
                return;
            }
            this.nodeRun = cloneDeep(s.workflowNodeRun);

            if (this.nodeRun && s.workflowRun) {
                this.historyLength = s.workflowRun.nodes[this.nodeRun.workflow_node_id].length;
                if (this.nodeRun.commits) {
                    this.commitsLength = this.nodeRun.commits.length;
                }
                if (this.nodeRun.artifacts) {
                    this.artifactLength = this.nodeRun.artifacts.length;
                }
                this.initVulnerabilitySummary();
                this.updateTitle(s.workflowRun);
            }
            this._cd.markForCheck();
        });

        this._activatedRoute.params.subscribe(params => {
            this._cd.markForCheck();
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

    updateTitle(wr: WorkflowRun) {
        if (!wr || !Array.isArray(wr.tags)) {
            return;
        }
        let branch = wr.tags.find((tag) => tag.tag === 'git.branch');
        if (branch) {
            this._titleService
                .setTitle(`Pipeline ${this.pipelineName} • #${wr.num} [${branch.value}] • ${this.workflowName}`);
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
