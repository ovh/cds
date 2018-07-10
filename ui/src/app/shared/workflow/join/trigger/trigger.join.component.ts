import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {finalize} from 'rxjs/operators';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {
    Workflow,
    WorkflowNode,
    WorkflowNodeCondition,
    WorkflowNodeConditions,
    WorkflowNodeContext,
    WorkflowNodeJoin,
    WorkflowNodeJoinTrigger
} from '../../../../model/workflow.model';
import {WorkflowNodeAddWizardComponent} from '../../../../shared/workflow/node/wizard/node.wizard.component';

@Component({
    selector: 'app-workflow-trigger-join',
    templateUrl: './workflow.trigger.join.html',
    styleUrls: ['./workflow.trigger.join.scss']
})
export class WorkflowTriggerJoinComponent {

    @ViewChild('triggerJoinModal')
    modalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @ViewChild('nodeWizard')
    nodeWizard: WorkflowNodeAddWizardComponent;

    @Output() triggerChange = new EventEmitter<WorkflowNodeJoinTrigger>();
    @Input() join: WorkflowNodeJoin;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() trigger: WorkflowNodeJoinTrigger;
    @Input() loading: boolean;

    currentSection = 'pipeline';

    constructor(private _modalService: SuiModalService) {
    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.modalTemplate);
        this.modal = this._modalService.open(config);
    }

    destNodeChange(node: WorkflowNode): void {
        this.trigger.workflow_dest_node = node;
    }

    saveTrigger(): void {
        this.loading = true;
        this.nodeWizard.goToNextSection()
          .pipe(finalize(() => this.loading = false))
          .subscribe(() => {
            if (!this.trigger.workflow_dest_node.id) {
                if (!this.trigger.workflow_dest_node.context) {
                    this.trigger.workflow_dest_node.context = new WorkflowNodeContext();
                }
                this.trigger.workflow_dest_node.context.conditions = new WorkflowNodeConditions();
                this.trigger.workflow_dest_node.context.conditions.plain = new Array<WorkflowNodeCondition>();
                let c = new  WorkflowNodeCondition();
                c.variable = 'cds.status';
                c.value = PipelineStatus.SUCCESS;
                c.operator = 'eq';
                this.trigger.workflow_dest_node.context.conditions.plain.push(c);
            }
            this.triggerChange.emit(this.trigger);
          });
    }

    pipelineSectionChanged(pipSection: string) {
        this.currentSection = pipSection;
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

    hide(): void {
        this.modal.approve(true);
    }
}
