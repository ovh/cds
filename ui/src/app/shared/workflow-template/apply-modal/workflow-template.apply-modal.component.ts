import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    Output,
    ViewChild
} from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Project } from 'app/model/project.model';
import { WorkflowTemplate, WorkflowTemplateInstance } from 'app/model/workflow-template.model';
import { Workflow } from 'app/model/workflow.model';
import { ProjectService } from 'app/service/project/project.service';
import { WorkflowTemplateService } from 'app/service/workflow-template/workflow-template.service';
import { calculateWorkflowTemplateDiff } from 'app/shared/diff/diff';
import { Item } from 'app/shared/diff/list/diff.list.component';
import { forkJoin } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-template-apply-modal',
    templateUrl: './workflow-template.apply-modal.html',
    styleUrls: ['./workflow-template.apply-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTemplateApplyModalComponent implements OnChanges {
    @ViewChild('workflowTemplateApplyModal', {static: false}) workflowTemplateApplyModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    open: boolean;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() workflowTemplate: WorkflowTemplate;
    @Input() workflowTemplateInstance: WorkflowTemplateInstance;
    @Output() close = new EventEmitter();

    diffVisible: boolean;
    diffItems: Array<Item>;
    workflowTemplateAuditMessages: Array<string>;

    constructor(
        private _modalService: SuiModalService,
        private _projectService: ProjectService,
        private _templateService: WorkflowTemplateService,
        private _cd: ChangeDetectorRef
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
        this.modal.onApprove(() => {
            this.diffVisible = false;
            this.open = false;
            this.close.emit();
        });
        this.modal.onDeny(() => {
            this.diffVisible = false;
            this.open = false;
            this.close.emit();
        });

        this.load();
    }

    load() {
        if (this.workflowTemplate && this.workflowTemplateInstance) {
            this._projectService.getProject(this.workflowTemplateInstance.project.key, null)
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(p => {
                this.project = p;
                this.loadAudits()
            });
            return
        } else if (this.workflow) {
            // retreive workflow template and instance from given workflow
            let s = this.workflow.from_template.split('/');

            forkJoin<WorkflowTemplate, WorkflowTemplateInstance> (
                this._templateService.get(s[0], s.splice(1, s.length - 1).join('/')),
                this._templateService.getInstance(this.workflow.project_key, this.workflow.name)
            ).subscribe(res => {
                this.workflowTemplate = res[0];
                this.workflowTemplateInstance = res[1];
                this.loadAudits();
                this._cd.markForCheck();
            });
        }
    }

    apply() {
        this._templateService.getInstance(this.workflowTemplateInstance.project.key,
            this.workflowTemplateInstance.workflow_name).subscribe(i => {
                this.workflowTemplateInstance = i;
            });
    }

    loadAudits() {
        // load audits since instance version if not latest
        if (this.workflowTemplateInstance.workflow_template_version !== this.workflowTemplate.version) {
            this._templateService.getAudits(this.workflowTemplate.group.name, this.workflowTemplate.slug,
                this.workflowTemplateInstance.workflow_template_version)
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(as => {
                    this.workflowTemplateAuditMessages = as.filter(a => !!a.change_message).map(a => a.change_message);
                    let before = as[as.length - 1].data_after ? as[as.length - 1].data_after : null;
                    this.diffItems = calculateWorkflowTemplateDiff(before, this.workflowTemplate);
                });
        } else {
            this.workflowTemplateAuditMessages = [];
            this.diffItems = [];
        }
    }

    clickClose() {
        this.modal.approve(true);
    }

    toggleDiff() {
        this.diffVisible = !this.diffVisible;
    }
}
