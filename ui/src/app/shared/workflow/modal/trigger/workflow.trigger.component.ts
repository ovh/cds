import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {finalize} from 'rxjs/operators';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {
    WNode, WNodeTrigger,
    Workflow, WorkflowNodeCondition, WorkflowNodeConditions
} from '../../../../model/workflow.model';
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

    @Output() triggerEvent = new EventEmitter<Workflow>();
    @Input() source: WNode;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() loading: boolean;
    @Input() destination: string;

    destNode: WNode;
    currentSection = 'pipeline';

    constructor(private _modalService: SuiModalService) {}

    show(): void {
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

    saveTrigger(): void {
        this.loading = true;
        this.nodeWizard.goToNextSection()
          .pipe(finalize(() => this.loading = false))
          .subscribe(() => {
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
