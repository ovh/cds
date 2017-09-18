import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeTrigger, WorkflowTriggerCondition} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui/src';
import {ActiveModal} from 'ng2-semantic-ui/src';

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

    operators: {};
    conditionNames: Array<string>;

    constructor(private _workflowStore: WorkflowStore, private _modalService: SuiModalService) {
    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.triggerModal);
        this.modal = this._modalService.open(config);
        this._workflowStore.getTriggerCondition(this.project.key, this.workflow.name, this.triggerSrcNode.id).first().subscribe( wtc => {
            this.operators = wtc.operators;
            this.conditionNames = wtc.names;
        });


    }

    hide(): void {
        this.modal.approve(true);
    }

    destNodeChange(node: WorkflowNode): void {
        this.trigger.workflow_dest_node = node;
    }

    saveTrigger(): void {
        this.triggerChange.emit(this.trigger);
    }

    addCondition(condition: WorkflowTriggerCondition): void {
        if (!this.trigger.conditions) {
            this.trigger.conditions = new Array<WorkflowTriggerCondition>();
        }
        let index = this.trigger.conditions.findIndex(c => c.variable === condition.variable);
        if (index === -1) {
            this.trigger.conditions.push(condition);
        }
    }
}
