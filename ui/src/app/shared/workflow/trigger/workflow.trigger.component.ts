import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {finalize} from 'rxjs/operators';
import {PipelineStatus} from '../../../model/pipeline.model';
import {Project} from '../../../model/project.model';
import {
    Workflow, WorkflowNode, WorkflowNodeCondition, WorkflowNodeConditions, WorkflowNodeContext
} from '../../../model/workflow.model';
import {WorkflowNodeAddWizardComponent} from '../../../shared/workflow/node/wizard/node.wizard.component';

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

    @Output() triggerChange = new EventEmitter<WorkflowNode>();
    @Input() source: string;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() trigger: WorkflowNode;
    @Input() loading: boolean;

    currentSection = 'pipeline';

    constructor(private _modalService: SuiModalService) {}

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.triggerModal);
        this.modal = this._modalService.open(config);
    }

    hide(): void {
        this.modal.approve(true);
    }

    destNodeChange(node: WorkflowNode): void {
        console.log(node);
        this.trigger = node;
    }

    pipelineSectionChanged(pipSection: string) {
        this.currentSection = pipSection;
    }

    saveTrigger(): void {
        this.loading = true;
        this.nodeWizard.goToNextSection()
          .pipe(finalize(() => this.loading = false))
          .subscribe(() => {
            if (!this.trigger.id) {
                if (!this.trigger.context) {
                    this.trigger.context = new WorkflowNodeContext();
                }
                this.trigger.context.conditions = new WorkflowNodeConditions();
                this.trigger.context.conditions.plain = new Array<WorkflowNodeCondition>();
                let c = new  WorkflowNodeCondition();
                c.variable = 'cds.status';
                c.value = PipelineStatus.SUCCESS;
                c.operator = 'eq';
                this.trigger.context.conditions.plain.push(c);
            }
            this.triggerChange.emit(this.trigger);
          });
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
