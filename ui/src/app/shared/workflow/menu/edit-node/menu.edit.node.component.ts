import {
    Component,
    EventEmitter,
    Input,
    Output
} from '@angular/core';
import {PermissionValue} from 'app/model/permission.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import {
    WNode,
    Workflow,
} from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {IPopup} from 'ng2-semantic-ui';

@Component({

    selector: 'app-workflow-menu-wnode-edit',
    templateUrl: './menu.edit.node.html',
    styleUrls: ['./menu.edit.node.scss'],
})
@AutoUnsubscribe()
export class WorkflowWNodeMenuEditComponent {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;
    _noderun: WorkflowNodeRun;
    @Input('noderun') set noderun(data: WorkflowNodeRun) {
        this._noderun = data;
        this.runnable = this.getCanBeRun();
    }
    get noderun() {return this._noderun}

    _workflowrun: WorkflowRun;
    @Input('workflowrun') set workflowrun(data: WorkflowRun) {
        this._workflowrun = data;
        this.runnable = this.getCanBeRun();
    }
    get workflowrun() { return this._workflowrun}

    @Input() popup: IPopup;
    @Input() readonly = true;
    @Output() event = new EventEmitter<string>();
    permissionEnum = PermissionValue;
    runnable: boolean;

    constructor() {}

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
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

            if (this.workflowrun) {
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
