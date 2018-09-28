import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {
    WNode, WNodeTrigger,
    Workflow, WorkflowNodeCondition, WorkflowNodeConditions
} from '../../../../model/workflow.model';
import {WorkflowNodeOutGoingHookFormComponent} from '../../node/outgoinghook-form/outgoinghook.form.component';
import {WorkflowNodeAddWizardComponent} from '../../node/wizard/node.wizard.component';

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

    constructor(private _modalService: SuiModalService) {}

    show(t: string): void {
        this.selectedType = t;
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

        if (this.source) {
            let clonedWorkflow = cloneDeep(this.workflow);
            let n = Workflow.getNodeByID(this.source.id, clonedWorkflow);
            if (!n.triggers) {
                n.triggers = new Array<WNodeTrigger>();
            }
            let newTrigger = new WNodeTrigger();
            newTrigger.parent_node_name = n.ref;
            newTrigger.child_node = this.destNode;
            n.triggers.push(newTrigger);
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
