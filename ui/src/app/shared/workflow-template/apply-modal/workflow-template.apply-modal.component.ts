import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter, inject,
    Input,
    OnChanges, OnInit,
    Output,
} from '@angular/core';
import { LoadOpts, Project } from 'app/model/project.model';
import { WorkflowTemplate, WorkflowTemplateInstance } from 'app/model/workflow-template.model';
import { Workflow } from 'app/model/workflow.model';
import { ProjectService } from 'app/service/project/project.service';
import { WorkflowTemplateService } from 'app/service/workflow-template/workflow-template.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { calculateWorkflowTemplateDiff } from 'app/shared/diff/diff';
import { Item } from 'app/shared/diff/list/diff.list.component';
import { forkJoin } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { NZ_MODAL_DATA, NzModalRef } from 'ng-zorro-antd/modal';

interface IModalData {
    workflowTemplateIn: WorkflowTemplate;
    workflowTemplateInstanceIn: WorkflowTemplateInstance
    projectIn: Project;
    workflowIn: Workflow;
}

@Component({
    selector: 'app-workflow-template-apply-modal',
    templateUrl: './workflow-template.apply-modal.html',
    styleUrls: ['./workflow-template.apply-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTemplateApplyModalComponent implements OnInit, OnChanges {

    // eslint-disable-next-line @angular-eslint/no-input-rename
    @Input('project') projectIn: Project;
    // eslint-disable-next-line @angular-eslint/no-input-rename
    @Input('workflow') workflowIn: Workflow;
    // eslint-disable-next-line @angular-eslint/no-input-rename
    @Input('workflowTemplate') workflowTemplateIn: WorkflowTemplate;
    // eslint-disable-next-line @angular-eslint/no-input-rename
    @Input('workflowTemplateInstance') workflowTemplateInstanceIn: WorkflowTemplateInstance;
    @Output() close = new EventEmitter();

    readonly nzModalData: IModalData = inject(NZ_MODAL_DATA);

    diffVisible: boolean;
    diffItems: Array<Item>;
    workflowTemplateAuditMessages: Array<string>;

    project: Project;
    workflow: Workflow;
    workflowTemplate: WorkflowTemplate;
    workflowTemplateInstance: WorkflowTemplateInstance;

    constructor(
        private _modal: NzModalRef,
        private _projectService: ProjectService,
        private _workflowService: WorkflowService,
        private _templateService: WorkflowTemplateService,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnChanges() {
        this.ngOnInit();
    }

    ngOnInit() {
        this.workflowTemplateIn = this.nzModalData.workflowTemplateIn;
        this.workflowTemplateInstanceIn = this.nzModalData.workflowTemplateInstanceIn;
        this.projectIn = this.nzModalData.projectIn;
        this.workflowIn = this.nzModalData.workflowIn;
        if (this.workflowTemplateIn && this.workflowTemplateInstanceIn) {
            this.workflowTemplate = this.workflowTemplateIn;
            this.workflowTemplateInstance = this.workflowTemplateInstanceIn;

            if (!this.workflowTemplateInstance.workflow) {
                this._projectService.getProject(this.workflowTemplateInstanceIn.project.key, [new LoadOpts('withKeys', 'keys')])
                    .pipe(finalize(() => this._cd.markForCheck()))
                    .subscribe(p => {
                        this.project = p;
                        this.loadAudits();
                        this._cd.markForCheck();
                    });
                return;
            }

            forkJoin([
                this._projectService.getProject(this.workflowTemplateInstanceIn.project.key, [new LoadOpts('withKeys', 'keys')]),
                this._workflowService.getWorkflow(this.workflowTemplateInstance.project.key, this.workflowTemplateInstance.workflow_name)
            ])
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(results => {
                    this.project = results[0];
                    this.workflow = results[1];
                    this.loadAudits();
                });
            return;
        } else if (this.projectIn && this.workflowIn) {
            // retrieve workflow template and instance from given workflow
            let s = this.workflowIn.from_template.split('@');
            s = s[0].split('/');
            this.project = this.projectIn;
            this.workflow = this.workflowIn;
            this.workflowTemplateInstance = this.workflowIn.template_instance;
            this._templateService.get(s[0], s.splice(1, s.length - 1).join('/'))
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(wt => {
                    this.workflowTemplate = wt;
                    this.loadAudits();
                });
        }
        this._cd.markForCheck();
    }

    onApply() {
        this._workflowService.getWorkflow(this.workflowTemplateInstance.request.project_key,
            this.workflowTemplateInstance.workflow_name).subscribe(w => {
                this.workflowTemplateInstance = Object.assign({}, w.template_instance);
                this._cd.markForCheck();
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
        this._modal.triggerOk();
    }

    toggleDiff() {
        this.diffVisible = !this.diffVisible;
    }
}
