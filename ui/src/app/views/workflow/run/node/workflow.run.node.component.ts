import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationExtras, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { RouterService } from 'app/service/router/router.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { AuthenticationState } from 'app/store/authentication.state';
import { ProjectState } from 'app/store/project.state';
import { GetWorkflowNodeRun, GetWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-run-node',
    templateUrl: './node.html',
    styleUrls: ['./node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeRunComponent implements OnInit {
    // Context info
    project: Project;
    project$: Subscription;
    workflowName: string;

    // data of the view
    currentNodeRunID: number;
    currentNodeRunStatus: string;
    currentNodeRunNum: number;
    commitsLength: number = 0;
    artifactLength: number = 0;
    staticFilesLength: number = 0;
    historyLength: number = 0;
    testsTotal: number = 0;
    hasVulnerability;

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
        let params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
        this.workflowName = params['workflowName'];

        let number = params['number'];
        let nodeRunId = params['nodeId'];

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

    ngOnInit(): void {
        this.storeSub = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            if (!s.workflow || this.workflowName !== s.workflow.name) {
                return;
            }
            if (!s.workflowNodeRun || !s.workflowRun) {
                return;
            }

            let refresh = false;

            if (s.workflowNodeRun && s.workflowRun) {
                if (!this.currentNodeRunID) {
                    this.currentNodeRunID = s.workflowNodeRun.id;
                    this.currentNodeRunNum = s.workflowNodeRun.num;
                    refresh = true;
                }

                if (this.currentNodeRunStatus !== s.workflowNodeRun.status) {
                    this.currentNodeRunStatus = s.workflowNodeRun.status;
                    refresh = true;
                }

                if (s.workflowRun.nodes[s.workflowNodeRun.workflow_node_id].length !== this.historyLength) {
                    this.historyLength = s.workflowRun.nodes[s.workflowNodeRun.workflow_node_id].length;
                    refresh = true;
                }


                if (s.workflowNodeRun.commits && s.workflowNodeRun.commits.length !== this.commitsLength) {
                    this.commitsLength = s.workflowNodeRun.commits.length;
                    refresh = true;
                }
                if (s.workflowNodeRun.artifacts && s.workflowNodeRun.artifacts.length !== this.artifactLength) {
                    this.artifactLength = s.workflowNodeRun.artifacts.length;
                    refresh = true;
                }
                if (s.workflowNodeRun.tests && s.workflowNodeRun.tests.total !== this.testsTotal) {
                    this.testsTotal = s.workflowNodeRun.tests.total;
                    refresh = true;
                }
                if (s.workflowNodeRun.vulnerabilities_report) {
                    this.hasVulnerability = true;
                    let result = this.initVulnerabilitySummary(s.workflowNodeRun);
                    if (this.nbVuln !== result['nbVuln']) {
                        this.nbVuln = result['nbVuln'];
                        refresh = true;
                    }
                    if (this.deltaVul !== result['deltaVuln']) {
                        this.deltaVul = result['deltaVuln'];
                        refresh = true;
                    }
                }
                if (s.workflowRun.tags) {
                    let branch = s.workflowRun.tags.find((tag) => tag.tag === 'git.branch');
                    if (branch) {
                        this._titleService
                            .setTitle(`Pipeline ${this.pipelineName} • #${s.workflowRun.num} [${branch.value}] • ${this.workflowName}`);
                    }
                }
            }
            if (refresh) {
                this._cd.markForCheck();
            } else {
                console.log('run:node:view:norefresh');
            }

        });

    }

    showTab(tab: string): void {
        let queryParams = Object.assign({}, this._activatedRoute.snapshot.queryParams, { tab })
        let navExtras: NavigationExtras = { queryParams };
        this._router.navigate(['project', this.project.key,
            'workflow', this.workflowName,
            'run', this.currentNodeRunNum,
            'node', this.currentNodeRunID], navExtras);
    }

    initVulnerabilitySummary(nodeRun: WorkflowNodeRun): any[] {
        let result = [];
        result['nbVuln'] = 0;
        result['deltaVuln'] = 0;
        if (nodeRun && nodeRun.vulnerabilities_report && nodeRun.vulnerabilities_report.report) {
            if (nodeRun.vulnerabilities_report.report.summary) {
                Object.keys(nodeRun.vulnerabilities_report.report.summary).forEach(k => {
                    result['nbVuln'] += nodeRun.vulnerabilities_report.report.summary[k];
                });
            }
            let previousNb = 0;
            if (nodeRun.vulnerabilities_report.report.previous_run_summary) {
                Object.keys(nodeRun.vulnerabilities_report.report.previous_run_summary).forEach(k => {
                    previousNb += nodeRun.vulnerabilities_report.report.previous_run_summary[k];
                });
            } else if (nodeRun.vulnerabilities_report.report.default_branch_summary) {
                Object.keys(nodeRun.vulnerabilities_report.report.default_branch_summary).forEach(k => {
                    previousNb += nodeRun.vulnerabilities_report.report.default_branch_summary[k];
                });
            }
            result['deltaVuln'] = this.nbVuln - previousNb;
        }
        return result;
    }
}
