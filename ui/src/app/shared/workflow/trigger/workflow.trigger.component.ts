import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {
    Workflow, WorkflowNode, WorkflowNodeCondition, WorkflowNodeConditions, WorkflowNodeContext, WorkflowNodeTrigger
} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-workflow-trigger',
    templateUrl: './workflow.trigger.html',
    styleUrls: ['./workflow.trigger.scss']
})
export class WorkflowTriggerComponent {

    @ViewChild('triggerModal')
    triggerModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Output() triggerChange = new EventEmitter<WorkflowNodeTrigger>();
    @Input() triggerSrcNode: WorkflowNode;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() trigger: WorkflowNodeTrigger;
    @Input() loading: boolean;

    constructor(private _modalService: SuiModalService) {
    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.triggerModal);
        this.modal = this._modalService.open(config);
    }

    hide(): void {
        this.modal.approve(true);
    }

    destNodeChange(node: WorkflowNode): void {
        this.trigger.workflow_dest_node = node;
    }

    saveTrigger(): void {
        if (!this.trigger.workflow_dest_node.id) {
            if (!this.trigger.workflow_dest_node.context) {
                this.trigger.workflow_dest_node.context = new WorkflowNodeContext();
            }
            this.trigger.workflow_dest_node.context.conditions = new WorkflowNodeConditions();
            this.trigger.workflow_dest_node.context.conditions.plain = new Array<WorkflowNodeCondition>();
            let c = new  WorkflowNodeCondition();
            c.variable = 'cds.status';
            c.value = 'Success';
            c.operator = 'eq';
            this.trigger.workflow_dest_node.context.conditions.plain.push(c);

        }
        this.triggerChange.emit(this.trigger);
    }
}
