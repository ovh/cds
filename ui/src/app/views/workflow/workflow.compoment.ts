import {Component, ViewChild} from '@angular/core';
import {SemanticSidebarComponent} from 'ng-semantic/ng-semantic';
import {ActivatedRoute, ResolveEnd, Router} from '@angular/router';
import {Project} from '../../model/project.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {Workflow, WorkflowNode, WorkflowNodeJoin} from '../../model/workflow.model';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {RouterService} from '../../service/router/router.service';
import {WorkflowCoreService} from '../../service/workflow/workflow.core.service';
import {finalize} from 'rxjs/operators';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowComponent {

    project: Project;
    workflow: Workflow;
    loading = true;
    number: number;
    workflowSubscription: Subscription;
    sideBarSubscription: Subscription;
    sidebarOpen: boolean;
    currentNodeName: string;
    selectedNodeId: number;
    selectedNode: WorkflowNode;
    selectedJoinId: number;
    selectedJoin: WorkflowNodeJoin;
    selectedNodeRunId: number;
    selectedNodeRunNum: number;

    @ViewChild('invertedSidebar')
    sidebar: SemanticSidebarComponent;

    constructor(private _activatedRoute: ActivatedRoute, private _workflowStore: WorkflowStore, private _router: Router,
                private _routerService: RouterService, private _workflowCore: WorkflowCoreService) {
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.sideBarSubscription = this._workflowCore.getSidebarStatus().subscribe(b => {
            this.sidebarOpen = b;
        });

        this._activatedRoute.params.subscribe(p => {
            let workflowName = p['workflowName'];
            this.number = p['number'];
            if (this.project.key && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }
                this.loading = true;
                this.workflowSubscription = this._workflowStore.getWorkflows(this.project.key, workflowName)
                    .subscribe(ws => {
                        if (ws) {
                            let updatedWorkflow = ws.get(this.project.key + '-' + workflowName);
                            if (updatedWorkflow && !updatedWorkflow.externalChange) {
                                this.workflow = updatedWorkflow;
                            }

                            if (this.selectedNodeId) {
                                this.selectedNode = this.findNode(this.workflow.root, this.workflow.joins);
                            } else if (this.selectedJoinId) {
                                this.selectedJoin = this.findJoin(this.workflow.joins);
                            }
                        }
                        this.loading = false;
                    }, () => {
                        this.loading = false;
                        this._router.navigate(['/project', this.project.key]);
                    });

            }

        });

        let snapshotparams = this._routerService.getRouteSnapshotParams({}, this._activatedRoute.snapshot);
        if (snapshotparams) {
            this.number = snapshotparams['number'];
        }
        let qp = this._routerService.getRouteSnapshotQueryParams({}, this._activatedRoute.snapshot);
        if (qp) {
            this.currentNodeName = qp['name'];
        }

        this._activatedRoute.queryParams.subscribe((queryp) => {
            if (queryp['selectedNodeId']) {
                this.selectedJoinId = null;
                this.selectedJoin = null;
                this.selectedNodeId = Number.isNaN(queryp['selectedNodeId']) ? null : parseInt(queryp['selectedNodeId'], 10);
            } else {
                this.selectedNodeId = null;
                this.selectedNode = null;
            }

            if (queryp['selectedJoinId']) {
                this.selectedNodeId = null;
                this.selectedNode = null;
                this.selectedJoinId = Number.isNaN(queryp['selectedJoinId']) ? null : parseInt(queryp['selectedJoinId'], 10);
            } else {
                this.selectedJoinId = null;
                this.selectedJoin = null;
            }

            if (queryp['selectedNodeRunId'] || queryp['selectedNodeRunNum']) {
                this.selectedJoinId = null;
                this.selectedJoin = null;
                this.selectedNodeRunId = Number.isNaN(queryp['selectedNodeRunId']) ? null : parseInt(queryp['selectedNodeRunId'], 10);
                this.selectedNodeRunNum = Number.isNaN(queryp['selectedNodeRunNum']) ? null : parseInt(queryp['selectedNodeRunNum'], 10);
            } else {
                this.selectedNodeRunId = null;
                this.selectedNodeRunNum = null;
            }

            if (this.selectedNodeId && !this.loading && this.workflow) {
                this.selectedNode = this.findNode(this.workflow.root, this.workflow.joins);
            } else if (this.selectedJoinId && !this.loading && this.workflow) {
                this.selectedJoin = this.findJoin(this.workflow.joins);
            }
        })

        this._router.events.subscribe(p => {
            if (p instanceof ResolveEnd) {
                let params = this._routerService.getRouteSnapshotParams({}, p.state.root);
                let queryParams = this._routerService.getRouteSnapshotQueryParams({}, p.state.root);
                this.currentNodeName = queryParams['name'];
                this.number = params['number'];
                if (qp['selectedNodeId']) {
                    this.selectedNodeId = Number.isNaN(qp['selectedNodeId']) ? null : parseInt(qp['selectedNodeId'], 10);
                }

                if (this.selectedNodeId && !this.loading) {
                    this.selectedNode = this.findNode(this.workflow.root, this.workflow.joins);
                } else if (this.selectedNodeId && !this.loading) {
                    this.selectedJoin = this.findJoin(this.workflow.joins);
                }
            }
        });
    }

    findNode(node: WorkflowNode, joins: WorkflowNodeJoin[]): WorkflowNode {
        let nodeFound;
        if (!node) {
            return;
        }

        if (this.selectedNodeId === node.id) {
            return node;
        }

        if (Array.isArray(node.triggers) && node.triggers.length) {
            for (let n of node.triggers) {
                nodeFound = this.findNode(n.workflow_dest_node, joins);
                if (nodeFound) {
                    return nodeFound;
                }
            }
        }

        if (!nodeFound && Array.isArray(joins) && joins.length) {
            for (let join of joins) {
                if (!Array.isArray(join.triggers) || !join.triggers.length) {
                    continue;
                }
                for (let tr of join.triggers) {
                    nodeFound = this.findNode(tr.workflow_dest_node, []);
                    if (nodeFound) {
                        return nodeFound;
                    }
                }
            }
        }

        return nodeFound;
    }

    findJoin(joins: WorkflowNodeJoin[]): WorkflowNodeJoin {
        if (!Array.isArray(joins)) {
            return null;
        }
        return joins.find((join) => join.id === this.selectedJoinId);
    }

    toggleSidebar(): void {
        this._workflowCore.moveSideBar(!this.sidebarOpen);
    }

    closeEditSidebar(): void {
        let qps = cloneDeep(this._activatedRoute.snapshot.queryParams);
        qps['selectedNodeId'] = null;
        qps['selectedJoinId'] = null;
        this.selectedNode = null;
        this.selectedNodeId = null;
        this.selectedJoin = null;
        this.selectedJoinId = null;
        this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name], {queryParams: qps});
    }
}
