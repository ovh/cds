import { Component, Input, OnChanges, ViewChild } from '@angular/core';
import { ModalTemplate, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal, SuiModalService } from 'ng2-semantic-ui/dist';
import { forkJoin } from 'rxjs';
import { Project } from '../../../model/project.model';
import { WorkflowTemplate, WorkflowTemplateInstance } from '../../../model/workflow-template.model';
import { Workflow } from '../../../model/workflow.model';
import { ProjectService } from '../../../service/project/project.service';
import { WorkflowTemplateService } from '../../../service/services.module';
import { calculateWorkflowTemplateDiff } from '../../diff/diff';
import { Item } from '../../diff/list/diff.list.component';

@Component({
    selector: 'app-workflow-template-apply-modal',
    templateUrl: './workflow-template.apply-modal.html',
    styleUrls: ['./workflow-template.apply-modal.scss']
})
export class WorkflowTemplateApplyModalComponent implements OnChanges {
    @ViewChild('workflowTemplateApplyModal') workflowTemplateApplyModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    open: boolean;

    _project: Project;
    @Input() set project(p: Project) { this._project = p; }
    get project(): Project { return this._project; }

    _workflow: Workflow;
    @Input() set workflow(w: Workflow) { this._workflow = w; }
    get workflow(): Workflow { return this._workflow; }

    _workflowTemplate: WorkflowTemplate;
    @Input() set workflowTemplate(wt: WorkflowTemplate) { this._workflowTemplate = wt; }
    get workflowTemplate(): WorkflowTemplate { return this._workflowTemplate; }

    _workflowTemplateInstance: WorkflowTemplateInstance;
    @Input() set workflowTemplateInstance(i: WorkflowTemplateInstance) { this._workflowTemplateInstance = i; }
    get workflowTemplateInstance(): WorkflowTemplateInstance { return this._workflowTemplateInstance; }

    diffVisible: boolean;
    diffItems: Array<Item>;
    workflowTemplateAuditMessages: Array<string>;

    constructor(
        private _modalService: SuiModalService,
        private _projectService: ProjectService,
        private _templateService: WorkflowTemplateService
    ) { }

    ngOnChanges() {
        if (this.open) {
            this.load();
        }
    }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.workflowTemplateApplyModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(() => { this.open = false; });
        this.modal.onDeny(() => { this.open = false; });

        this.load();
    }

    load() {
        if (this.workflowTemplate && this.workflowTemplateInstance) {
            this._projectService.getProject(this.workflowTemplateInstance.project.key, null).subscribe(p => {
                this._project = p;
                this.loadAudits()
            });
            return
        } else if (this.workflow) {
            // retreive workflow template and instance from given workflow
            let s = this.workflow.from_template.split('/');

            forkJoin(
                this._templateService.get(s[0], s.splice(1, s.length - 1).join('/')),
                this._templateService.getInstance(this.workflow.project_key, this.workflow.name)
            ).subscribe(res => {
                this._workflowTemplate = res[0];
                this._workflowTemplateInstance = res[1];
                this.loadAudits();
            });
        }
    }

    loadAudits() {
        // load audits since instance version if not latest
        if (this.workflowTemplateInstance.workflow_template_version !== this.workflowTemplate.version) {
            this._templateService.getAudits(this.workflowTemplate.group.name, this.workflowTemplate.slug,
                this.workflowTemplateInstance.workflow_template_version).subscribe(as => {
                    this.workflowTemplateAuditMessages = as.filter(a => !!a.change_message).map(a => a.change_message);
                    let before = as[as.length - 1].data_after ? <WorkflowTemplate>JSON.parse(as[as.length - 1].data_after) : null;
                    this.diffItems = calculateWorkflowTemplateDiff(before, this.workflowTemplate);
                });
        } else {
            this.workflowTemplateAuditMessages = [];
            this.diffItems = [];
        }
    }

    close() {
        this.diffVisible = false;
        this.modal.approve(true);
    }

    toggleDiff() {
        this.diffVisible = !this.diffVisible;
    }
}
