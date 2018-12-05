import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { cloneDeep } from 'lodash';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { PipelineStatus } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { WNode, WNodeTrigger, Workflow, WorkflowNodeCondition, WorkflowNodeConditions } from '../../../../model/workflow.model';
import { WorkflowNodeOutGoingHookFormComponent } from '../../node/outgoinghook-form/outgoinghook.form.component';
import { WorkflowNodeAddWizardComponent } from '../../node/wizard/node.wizard.component';

@Component({
    selector: 'app-workflow-trigger',
    templateUrl: './workflow.trigger.html',
    styleUrls: ['./workflow.trigger.scss']
})
export class WorkflowTriggerComponent {

    @ViewChild('triggerModal')
    triggerModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    @ViewChild('nodeWizard')
    nodeWizard: WorkflowNodeAddWizardComponent;
    @ViewChild('worklflowAddOutgoingHook')
    worklflowAddOutgoingHook: WorkflowNodeOutGoingHookFormComponent;

    @Output() triggerEvent = new EventEmitter<Workflow>();
    @Input() source: WNode;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() loading: boolean;
    @Input() destination: string;

    destNode: WNode;
    currentSection = 'pipeline';
    selectedType: string;
    isParent: boolean;

    constructor(private _modalService: SuiModalService) {}

    show(t: string, isP: boolean): void {
        this.selectedType = t;
        this.isParent = isP;
        const config = new TemplateModalConfig<boolean, boolean, void>(this.triggerModal);
        this.modal = this._modalService.open(config);
    }

    hide(): void {
        this.modal.approve(true);
    }

    destNodeChange(node: WNode): void {
        this.destNode = node;
    }

    pipelineSectionChanged(pipSection: string) {
        this.currentSection = pipSection;
    }

    addOutgoingHook(): void {
        this.destNode = this.worklflowAddOutgoingHook.hook;
        this.saveTrigger();
    }

    saveTrigger(): void {
        this.destNode.context.conditions = new WorkflowNodeConditions();
        this.destNode.context.conditions.plain = new Array<WorkflowNodeCondition>();
        let c = new  WorkflowNodeCondition();
        c.variable = 'cds.status';
        c.value = PipelineStatus.SUCCESS;
        c.operator = 'eq';
        this.destNode.context.conditions.plain.push(c);

        let clonedWorkflow = cloneDeep(this.workflow);
        if (this.source && !this.isParent) {
            let sourceNode = Workflow.getNodeByID(this.source.id, clonedWorkflow);
            if (!sourceNode.triggers) {
                sourceNode.triggers = new Array<WNodeTrigger>();
            }
            let newTrigger = new WNodeTrigger();
            newTrigger.parent_node_name = sourceNode.ref;
            newTrigger.child_node = this.destNode;
            sourceNode.triggers.push(newTrigger);
            this.triggerEvent.emit(clonedWorkflow);
        } else if (this.isParent) {
            this.destNode.triggers = new Array<WNodeTrigger>();
            let newTrigger = new WNodeTrigger();
            newTrigger.child_node = clonedWorkflow.workflow_data.node;
            this.destNode.triggers.push(newTrigger);
            this.destNode.context.default_payload = newTrigger.child_node.context.default_payload;
            newTrigger.child_node.context.default_payload = null;
            this.destNode.hooks = cloneDeep(clonedWorkflow.workflow_data.node.hooks);
            clonedWorkflow.workflow_data.node.hooks = [];
            clonedWorkflow.workflow_data.node = this.destNode;
            this.triggerEvent.emit(clonedWorkflow);
        }
    }

    nextStep() {
      this.nodeWizard.goToNextSection().subscribe((section) => {
        if (section === 'done') {
          this.saveTrigger();
        } else {
          this.currentSection = section;
        }
      });
    }
}
