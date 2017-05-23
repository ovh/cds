import {AfterViewInit, Component, ElementRef, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeTrigger} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';
import {WorkflowTriggerComponent} from '../trigger/workflow.trigger.component';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../toast/ToastService';
import {SemanticModalComponent} from 'ng-semantic';
import {WorkflowDeleteNodeComponent} from './delete/workflow.node.delete.component';


declare var _: any;
@Component({
    selector: 'app-workflow-node',
    templateUrl: './workflow.node.html',
    styleUrls: ['./workflow.node.scss']
})
export class WorkflowNodeComponent implements AfterViewInit {

    @Input() node: WorkflowNode;
    @Input() workflow: Workflow;
    @Input() project: Project;

    @ViewChild('workflowTrigger')
    workflowTrigger: WorkflowTriggerComponent;
    @ViewChild('workflowDeleteNode')
    workflowDeleteNode: WorkflowDeleteNodeComponent;

    newTrigger: WorkflowNodeTrigger = new WorkflowNodeTrigger();

    constructor(private elementRef: ElementRef, private _workflowStore: WorkflowStore, private _translate: TranslateService,
        private _toast: ToastService) {
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = 0;
    }

    openTriggerModal(): void {
        this.newTrigger = new WorkflowNodeTrigger();
        this.newTrigger.workflow_node_id = this.node.id;
        this.workflowTrigger.show({observable: true, closable: false, autofocus: false});
    }

    openDeleteNodeModal(): void {
        this.workflowDeleteNode.show({observable: true, closable: false, autofocus: false});
    }

    saveTrigger(): void {
        let clonedWorkflow: Workflow = _.cloneDeep(this.workflow);
        let currentNode: WorkflowNode;
        if (clonedWorkflow.root.id === this.node.id) {
            currentNode = clonedWorkflow.root;
        } else if (clonedWorkflow.root.triggers) {
            currentNode = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        }

        if (!currentNode) {
            return;
        }

        if (!currentNode.triggers) {
            currentNode.triggers = new Array<WorkflowNodeTrigger>();
        }
        currentNode.triggers.push(_.cloneDeep(this.newTrigger));
        this.updateWorkflow(clonedWorkflow, this.workflowTrigger.modal);
    }

    deleteNode(b: boolean): void {
        if (b) {
            let clonedWorkflow: Workflow = _.cloneDeep(this.workflow);
            if (clonedWorkflow.root.id === this.node.id) {
                this.deleteWorkflow(clonedWorkflow, this.workflowDeleteNode.modal);
            } else {
                clonedWorkflow.root.triggers.forEach((t, i) => {
                    this.removeNode(this.node.id, t.workflow_dest_node, clonedWorkflow.root, i);
                });
                this.updateWorkflow(clonedWorkflow, this.workflowDeleteNode.modal);
            }
        }
    }

    removeNode(id: number, node: WorkflowNode, parent: WorkflowNode, index: number) {
        if (node.id === id) {
            parent.triggers.splice(index, 1);
        }
        if (node.triggers) {
            node.triggers.forEach((t, i) => {
                this.removeNode(id, t.workflow_dest_node, node, i);
            });
        }
    }

    deleteWorkflow(w: Workflow, modal: SemanticModalComponent): void {
        this._workflowStore.deleteWorkflow(this.project.key, w).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_deleted'));
            modal.hide();
        });
    }

   updateWorkflow(w: Workflow, modal: SemanticModalComponent): void {
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
           this._toast.success('', this._translate.instant('workflow_updated'));
           modal.hide();
        });
    }
}
