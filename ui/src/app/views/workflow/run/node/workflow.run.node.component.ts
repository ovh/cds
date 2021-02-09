import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationExtras, Router } from '@angular/router';
import { Select, Store } from '@ngxs/store';
import { Tests } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { RouterService } from 'app/service/router/router.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { GetWorkflowNodeRun, GetWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-run-node',
    templateUrl: './node.html',
    styleUrls: ['./node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeRunComponent implements OnInit, OnDestroy {
    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;

    // Context info
    project: Project;
    project$: Subscription;
    workflowName: string;

    // data of the view
    currentNodeRunID: number;
    currentNodeRunNum: number;
    commitsLength = 0;
    artifactLength = 0;
    staticFilesLength = 0;
    historyLength = 0;
    nodeRunTests: Tests;
    hasVulnerability = false;

    pipelineName = '';

    // History
    nodeRunsHistory = new Array<WorkflowNodeRun>();
    selectedTab: string;

    nbVuln = 0;
    deltaVul = 0;

    paramsSub: Subscription;

    constructor(
        private _store: Store,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _routerService: RouterService,
        private _titleService: Title,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);

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

        this.paramsSub = this._activatedRoute.params.subscribe(p => {
            if (p['nodeId'] === this.currentNodeRunID && p['number'] === this.currentNodeRunNum) {
                return;
            }
            this._store.dispatch(new GetWorkflowRun({ projectKey: this.project.key, workflowName: this.workflowName, num: p['number'] }))
                .subscribe(() => {
                    this._store.dispatch(
                        new GetWorkflowNodeRun({
                            projectKey: this.project.key,
                            workflowName: this.workflowName,
                            num: p['number'],
                            nodeRunID: p['nodeId']
                        }));
                });
        });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
            let w = this._store.selectSnapshot(WorkflowState.workflowSnapshot);
            let wr = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowRun;
            if (!w || this.workflowName !== w.name) {
                return;
            }
            if (!nr || !wr) {
                return;
            }

            let refresh = false;

            if (nr && wr) {
                if (!this.currentNodeRunID) {
                    this.currentNodeRunID = nr.id;
                    this.currentNodeRunNum = nr.num;
                    refresh = true;
                }

                if (wr.nodes[nr.workflow_node_id].length !== this.historyLength) {
                    this.historyLength = wr.nodes[nr.workflow_node_id].length;
                    refresh = true;
                }


                if (nr.commits && nr.commits.length !== this.commitsLength) {
                    this.commitsLength = nr.commits.length;
                    refresh = true;
                }

                let artiResults = nr?.results?.filter(r => r.type === 'artifact');
                let artiResultsLength = 0;
                if (artiResults) {
                    artiResultsLength = artiResults.length
                }
                let oldArtiLength = 0;
                if (nr.artifacts) {
                    oldArtiLength = nr.artifacts.length;
                }
                if ((nr.artifacts || artiResults) && (oldArtiLength + artiResultsLength) !== this.artifactLength) {
                    this.artifactLength = oldArtiLength + artiResultsLength;
                    refresh = true;
                }
                if (nr.tests && nr.tests.total !== this.nodeRunTests?.total) {
                    this.nodeRunTests = nr.tests;
                    refresh = true;
                }
                if (nr.vulnerabilities_report && nr.vulnerabilities_report.id !== 0) {
                    this.hasVulnerability = true;
                    let result = this.initVulnerabilitySummary(nr);
                    if (this.nbVuln !== result['nbVuln']) {
                        this.nbVuln = result['nbVuln'];
                        refresh = true;
                    }
                    if (this.deltaVul !== result['deltaVuln']) {
                        this.deltaVul = result['deltaVuln'];
                        refresh = true;
                    }
                }
                if (wr.tags) {
                    let branch = wr.tags.find((tag) => tag.tag === 'git.branch');
                    if (branch) {
                        this._titleService
                            .setTitle(`Pipeline ${this.pipelineName} • #${nr.num} [${branch.value}] • ${this.workflowName}`);
                    }
                }
            }
            if (refresh) {
                this._cd.markForCheck();
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
