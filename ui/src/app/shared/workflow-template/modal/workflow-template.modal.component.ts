import { Component, Input, ViewChild } from '@angular/core';
import { ModalTemplate, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal, SuiModalService } from 'ng2-semantic-ui/dist';
import { Project } from '../../../model/project.model';
import { WorkflowTemplate, WorkflowTemplateInstance } from '../../../model/workflow-template.model';
import { Workflow } from '../../../model/workflow.model';
import { WorkflowTemplateService } from '../../../service/services.module';
import { Item } from '../../../shared/diff/list/diff.list.component';

@Component({
    selector: 'app-workflow-template-modal',
    templateUrl: './workflow-template.modal.html',
    styleUrls: ['./workflow-template.modal.scss']
})
export class WorkflowTemplateModalComponent {
    @ViewChild('workflowTemplateModal') workflowTemplateModal: ModalTemplate<boolean, boolean, void>;
    @Input() project: Project;
    @Input() workflow: Workflow;
    modal: ActiveModal<boolean, boolean, void>;
    workflowTemplate: WorkflowTemplate;
    workflowTemplateInstance: WorkflowTemplateInstance;
    diffItems: Array<Item>;

    constructor(
        private _modalService: SuiModalService,
        private _templateService: WorkflowTemplateService
    ) { }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.workflowTemplateModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);

        this.loadTemplate();
        this.loadInstance();
    }

    loadTemplate() {
        let s = this.workflow.from_template.split('/');
        if (s.length > 1) {
            this._templateService.getWorkflowTemplate(s[0], s.splice(1, s.length - 1).join('/')).subscribe(wt => {
                this.workflowTemplate = wt;
            });
        }
    }

    loadInstance() {
        this._templateService.getWorkflowTemplateInstance(this.project.key, this.workflow.name).subscribe(wti => {
            this.workflowTemplateInstance = wti;
        });
    }

    close() {
        this.modal.approve(true);
    }
}
