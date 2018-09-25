import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs';
import {PermissionValue} from '../../../../model/permission.model';
import {Project} from '../../../../model/project.model';
import {WNode, Workflow} from '../../../../model/workflow.model';
import {PipelineStore, WorkflowEventStore} from '../../../../service/services.module';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {WorkflowNodeConditionsComponent} from '../../modal/conditions/node.conditions.component';
import {WorkflowNodeContextComponent} from '../../modal/context/workflow.node.context.component';
import {WorkflowDeleteNodeComponent} from '../../modal/delete/workflow.node.delete.component';

@Component({
    selector: 'app-workflow-sidebar-wnode-edit',
    templateUrl: './sidebar.edit.html',
    styleUrls: ['./sidebar.edit.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeSidebarEditComponent implements OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    node: WNode;
    nodeSub: Subscription;

    displayInputName = false;
    previousNodeName: string;

    permissionEnum = PermissionValue;
    loading = false;

    // Modal
    @ViewChild('workflowDeleteNode')
    workflowDeleteNode: WorkflowDeleteNodeComponent;
    @ViewChild('workflowContext')
    workflowContext: WorkflowNodeContextComponent;
    @ViewChild('workflowConditions')
    workflowConditions: WorkflowNodeConditionsComponent;

    // Subscription
    pipelineSubscription: Subscription;

    constructor(private _workflowEventStore: WorkflowEventStore, private _pipelineStore: PipelineStore) {

    }

    ngOnInit(): void {
        this.nodeSub = this._workflowEventStore.selectedNode().subscribe(n => {
            if (n) {
                if (!this.displayInputName) {
                    this.previousNodeName = n.name
                }
            }
            this.node = n;
        });
    }

    canEdit(): boolean {
        return this.workflow.permission === PermissionValue.READ_WRITE_EXECUTE;
    }

    rename(): void {
        if (!this.canEdit()) {
            return;
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let node = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        if (!node) {
            return;
        }
        node.name = this.node.name;
        this.updateWorkflow(clonedWorkflow);
    }

    openDeleteNodeModal(): void {
        if (!this.canEdit()) {
            return;
        }
        if (this.workflowDeleteNode) {
            this.workflowDeleteNode.show();
        }
    }

    openEditContextModal(): void {
       this.pipelineSubscription =
            this._pipelineStore.getPipelines(this.project.key,
                this.workflow.pipelines[this.node.context.pipeline_id].name).subscribe(pips => {
                if (pips.get(this.project.key + '-' + this.workflow.pipelines[this.node.context.pipeline_id].name)) {
                    setTimeout(() => {
                        this.workflowContext.show();
                        this.pipelineSubscription.unsubscribe();
                    }, 100);
                }
            });
    }

    openEditRunConditions(): void {
        this.workflowConditions.show();
    }

    updateWorkflow(w: Workflow): void {
        console.log(w);
    }
}
