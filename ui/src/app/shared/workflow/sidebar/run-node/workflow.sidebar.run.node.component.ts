import { Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import {Store} from '@ngxs/store';
import {WorkflowState, WorkflowStateModel} from 'app/store/workflow.state';
import { Subscription } from 'rxjs';
import 'rxjs/add/observable/zip';
import { first } from 'rxjs/operators';
import { PermissionValue } from '../../../../model/permission.model';
import { PipelineStatus } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { WNode, WNodeType, Workflow } from '../../../../model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from '../../../../model/workflow.run.model';
import { WorkflowRunService } from '../../../../service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { DurationService } from '../../../duration/duration.service';
import { WorkflowNodeRunParamComponent } from '../../node/run/node.run.param.component';


@Component({
    selector: 'app-workflow-sidebar-run-node',
    templateUrl: './workflow.sidebar.run.node.component.html',
    styleUrls: ['./workflow.sidebar.run.node.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunNodeComponent implements OnDestroy, OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    currentWorkflowRun: WorkflowRun;
    currentWorkflowNodeRun: WorkflowNodeRun;
    storeSub: Subscription;

    node: WNode;
    subNode: Subscription;
    wNodeType = WNodeType;

    // Modal
    @ViewChild('workflowRunNode')
    workflowRunNode: WorkflowNodeRunParamComponent;
    loading = true;

    displayEditOption = false;
    duration: string;
    canBeRun = false;
    perm = PermissionValue;
    pipelineStatusEnum = PipelineStatus;

    durationIntervalID: number;

    constructor(
        private _wrService: WorkflowRunService,
        private _router: Router,
        private _durationService: DurationService,
        private _store: Store
    ) { }

    ngOnInit(): void {
        this.storeSub = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            this.currentWorkflowRun = s.workflowRun;
            this.node = s.node;
            this.currentWorkflowNodeRun = s.workflowNodeRun;
            if (!s.workflowRun) {
                return;
            }
            if (this.node && this.loading) {
                this.loading = false;
            } else if (!this.node) {
                this.loading = true;
            }
            this.refreshData();

            if (!s.workflowNodeRun) {
                return;
            }
            this.loading = false;
            this.deleteInverval();
            this.duration = this.getDuration();
            this.durationIntervalID = window.setInterval(() => {
                this.duration = this.getDuration();
            }, 5000);

            this.refreshData();
        });
    }

    refreshData(): void {
        this.displayEditOption = this.node != null;
        this.canBeRun = this.getCanBeRun();
        // TODO REMOVE
        if (this.node && (this.node.type === WNodeType.FORK || this.node.type === WNodeType.OUTGOINGHOOK)
            && ((this.currentWorkflowRun && this.currentWorkflowRun.version < 2) || !this.currentWorkflowRun)) {
            this.canBeRun = false;
        }
    }

    displayLogs() {
        switch (this.node.type) {
            case WNodeType.OUTGOINGHOOK:
                if (this.currentWorkflowNodeRun && this.node && this.node.outgoing_hook
                    && this.node.outgoing_hook.config['target_workflow']) {
                    this._router.navigate([
                        '/project', this.project.key,
                        'workflow', this.node.outgoing_hook.config['target_workflow'].value,
                        'run', this.currentWorkflowNodeRun.callback.workflow_run_number
                    ], { queryParams: {} });
                }
                break;
            default:
                this._router.navigate([
                    '/project', this.project.key,
                    'workflow', this.workflow.name,
                    'run', this.currentWorkflowRun.num,
                    'node', this.currentWorkflowNodeRun.id], { queryParams: { name: this.node.name } });
        }
    }

    getDuration() {
        if (!this.currentWorkflowNodeRun) {
            return;
        }
        let done = new Date(this.currentWorkflowNodeRun.done);
        if (PipelineStatus.isActive(this.currentWorkflowNodeRun.status)) {
            done = new Date();
        } else {
            this.deleteInverval();
        }

        return this._durationService.duration(new Date(this.currentWorkflowNodeRun.start), done);
    }

    getCanBeRun(): boolean {
        if (!this.workflow) {
            return;
        }

        // Get Env permission
        let envForbid = this.node && this.node.context && this.node.context.environment_id > 1
            && this.workflow.environments && this.workflow.environments[this.node.context.environment_id]
            && this.workflow.environments[this.node.context.environment_id].permission
            && this.workflow.environments[this.node.context.environment_id].permission < PermissionValue.READ_EXECUTE;

        if (this.workflow && this.workflow.permission < PermissionValue.READ_EXECUTE || envForbid) {
            return false;
        }

        // If we are in a run, check if current node can be run ( compuite by cds api)
        if (this.currentWorkflowNodeRun && this.currentWorkflowRun && this.currentWorkflowRun.nodes) {
            let nodesRun = this.currentWorkflowRun.nodes[this.currentWorkflowNodeRun.workflow_node_id];
            if (nodesRun) {
                let nodeRun = nodesRun.find(n => {
                    return n.id === this.currentWorkflowNodeRun.id;
                });
                if (nodeRun) {
                    return nodeRun.can_be_run;
                }
            }
            return false;
        }

        let workflowRunIsNotActive = this.currentWorkflowRun && !PipelineStatus.isActive(this.currentWorkflowRun.status);
        if (workflowRunIsNotActive && this.currentWorkflowNodeRun) {
            return true;
        }

        if (this.node && this.currentWorkflowRun) {
            if (workflowRunIsNotActive && !this.currentWorkflowNodeRun &&
                this.node.id === this.currentWorkflowRun.workflow.workflow_data.node.id) {
                return true;
            }

            if (this.currentWorkflowRun) {
                let nbNodeFound = 0;
                let parentNodes = Workflow.getParentNodeIds(this.currentWorkflowRun, this.node.id);
                for (let parentNodeId of parentNodes) {
                    for (let nodeRunId in this.currentWorkflowRun.nodes) {
                        if (!this.currentWorkflowRun.nodes[nodeRunId]) {
                            continue;
                        }
                        let nodeRuns = this.currentWorkflowRun.nodes[nodeRunId];
                        if (nodeRuns[0].workflow_node_id === parentNodeId) { // if node id is still the same
                            if (PipelineStatus.isActive(nodeRuns[0].status)) {
                                return false;
                            }
                            nbNodeFound++;
                        } else if (!Workflow.getNodeByID(nodeRuns[0].workflow_node_id, this.currentWorkflowRun.workflow)) {
                            // workflow updated so prefer return true
                            return true;
                        }
                    }
                }
                if (nbNodeFound !== parentNodes.length) { // It means that a parent node isn't already executed
                    return false;
                }
            }
        }
        return true;
    }

    stopNodeRun(): void {
        this.loading = true;
        this._wrService.stopNodeRun(this.project.key, this.workflow.name,
            this.currentWorkflowRun.num, this.currentWorkflowNodeRun.id)
            .pipe(first())
            .subscribe(() => {
                this._router.navigate([
                    '/project', this.project.key,
                    'workflow', this.workflow.name,
                    'run', this.currentWorkflowRun.num]);
            });
    }

    openRunNode(): void {
        this.workflowRunNode.show();
    }

    ngOnDestroy(): void {
        this.deleteInverval();
    }

    deleteInverval(): void {
        if (this.durationIntervalID) {
            clearInterval(this.durationIntervalID);
            this.durationIntervalID = 0;
        }
    }
}
