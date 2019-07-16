import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output
} from '@angular/core';
import { Store } from '@ngxs/store';
import { IPopup } from '@richardlt/ng2-semantic-ui';
import { PermissionValue } from 'app/model/permission.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-menu-wnode-edit',
    templateUrl: './menu.edit.node.html',
    styleUrls: ['./menu.edit.node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeMenuEditComponent implements OnInit {

    // Project that contains the workflow
    @Input() project: Project;

    @Input() node: WNode;
    _noderun: WorkflowNodeRun;
    @Input('noderun') set noderun(data: WorkflowNodeRun) {
        this._noderun = data;
        this.runnable = this.getCanBeRun();
    }
    get noderun() { return this._noderun }

    _workflowrun: WorkflowRun;
    @Input('workflowrun') set workflowrun(data: WorkflowRun) {
        this._workflowrun = data;
        this.runnable = this.getCanBeRun();
    }
    get workflowrun() { return this._workflowrun }

    @Input() popup: IPopup;
    @Input() readonly = true;
    @Output() event = new EventEmitter<string>();
    permissionEnum = PermissionValue;
    runnable: boolean;
    storeSubscription: Subscription;
    workflow: Workflow;

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        this.storeSubscription = this._store.select(WorkflowState.getCurrent())
            .subscribe((s: WorkflowStateModel) => {
            this.workflow = s.workflow;
            this.runnable = this.getCanBeRun();
            this._cd.markForCheck();
        });
    }

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
    }

    getCanBeRun(): boolean {
        if (!this.workflow) {
            return;
        }

        if (this.workflow && !this.workflow.permissions.executable) {
            return false;
        }

        // If we are in a run, check if current node can be run ( compuite by cds api)
        if (this.noderun && this.workflowrun && this.workflowrun.nodes) {
            let nodesRun = this.workflowrun.nodes[this.noderun.workflow_node_id];
            if (nodesRun) {
                let nodeRun = nodesRun.find(n => {
                    return n.id === this.noderun.id;
                });
                if (nodeRun) {
                    return nodeRun.can_be_run;
                }
            }
            return false;
        }

        let workflowrunIsNotActive = this.workflowrun && !PipelineStatus.isActive(this.workflowrun.status);
        if (workflowrunIsNotActive && this.noderun) {
            return true;
        }

        if (this.node && this.workflowrun) {
            if (workflowrunIsNotActive && !this.noderun &&
                this.node.id === this.workflowrun.workflow.workflow_data.node.id) {
                return true;
            }

            if (this.workflowrun && this.workflowrun.workflow && this.workflowrun.workflow.workflow_data) {
                let nbNodeFound = 0;
                let parentNodes = Workflow.getParentNodeIds(this.workflowrun, this.node.id);
                for (let parentNodeId of parentNodes) {
                    for (let nodeRunId in this.workflowrun.nodes) {
                        if (!this.workflowrun.nodes[nodeRunId]) {
                            continue;
                        }
                        let nodeRuns = this.workflowrun.nodes[nodeRunId];
                        if (nodeRuns[0].workflow_node_id === parentNodeId) { // if node id is still the same
                            if (PipelineStatus.isActive(nodeRuns[0].status)) {
                                return false;
                            }
                            nbNodeFound++;
                        } else if (!Workflow.getNodeByID(nodeRuns[0].workflow_node_id, this.workflowrun.workflow)) {
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
}
