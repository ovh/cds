import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { DeleteWorkflow, DeleteWorkflowIcon, UpdateWorkflow, UpdateWorkflowIcon } from 'app/store/workflows.action';
import { cloneDeep } from 'lodash';
import { finalize, first } from 'rxjs/operators';
import { Project } from '../../../../model/project.model';
import { Workflow } from '../../../../model/workflow.model';
import { WorkflowRunService } from '../../../../service/workflow/run/workflow.run.service';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../shared/toast/ToastService';


@Component({
    selector: 'app-workflow-admin',
    templateUrl: 'workflow.admin.component.html',
    styleUrls: ['./workflow.admin.scss']
})

export class WorkflowAdminComponent implements OnInit {

    @Input() project: Project;

    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = cloneDeep(data);
            if (this._workflow.purge_tags && this._workflow.purge_tags.length) {
                this.purgeTag = this._workflow.purge_tags[0];
            }
        }
    };
    get workflow() { return this._workflow };

    oldName: string;

    runnumber: number;

    existingTags = new Array<string>();
    selectedTags = new Array<string>();
    purgeTag: string;
    iconUpdated = false;

    @ViewChild('updateWarning')
    private warningUpdateModal: WarningModalComponent;

    loading = false;
    fileTooLarge = false;

    constructor(
        private store: Store,
        public _translate: TranslateService,
        private _toast: ToastService,
        private _router: Router,
        private _workflowRunService: WorkflowRunService
    ) { }

    ngOnInit(): void {
        if (!this._workflow.metadata) {
            this._workflow.metadata = new Map<string, string>();
        }
        if (this._workflow.metadata['default_tags']) {
            this.selectedTags = this._workflow.metadata['default_tags'].split(',');
            this.existingTags.push(...this.selectedTags);
        }

        if (this.project.permission !== 7) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }
        this.oldName = this.workflow.name;

        this._workflowRunService.getTags(this.project.key, this._workflow.name).subscribe(tags => {
            let existingTags = [];
            Object.keys(tags).forEach(k => {
                if (tags.hasOwnProperty(k) && this.existingTags.indexOf(k) === -1) {
                    existingTags.push(k);
                }
            });
            this.existingTags = this.existingTags.concat(existingTags);
        });
        this._workflowRunService.getRunNumber(this.project.key, this.workflow).pipe(first()).subscribe(n => {
            this.runnumber = n.num;
        });
    }

    updateWorkflow(): void {
        this.loading = true;
        this._workflow.purge_tags = [this.purgeTag];
        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            changes: this.workflow
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }

    deleteIcon(): void {
        this.loading = true;
        this.store.dispatch(new DeleteWorkflowIcon({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }

    updateIcon(): void {
        this.loading = true;
        this.store.dispatch(new UpdateWorkflowIcon({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            icon: this.workflow.icon
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.iconUpdated = false;
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }

    updateTagMetadata(m): void {
        this._workflow.metadata['default_tags'] = m.join(',');
    }

    onSubmitWorkflowRunNumUpdate() {
        this.loading = true;

        this._workflowRunService.updateRunNumber(this.project.key, this.workflow, this.runnumber).pipe(first(), finalize(() => {
            this.loading = false;
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
        });
    }

    onSubmitWorkflowUpdate(skip?: boolean) {
        if (!skip && this.workflow.externalChange) {
            this.warningUpdateModal.show();
        } else {
            this.loading = true;
            this.store.dispatch(new UpdateWorkflow({
                projectKey: this.project.key,
                workflowName: this.oldName,
                changes: this.workflow
            })).pipe(finalize(() => this.loading = false))
                .subscribe(() => {
                    this._toast.success('', this._translate.instant('workflow_updated'));
                    this._router.navigate([
                        '/project', this.project.key, 'workflow', this.workflow.name
                    ], { queryParams: { tab: 'advanced' } });
                });
        }
    };

    deleteWorkflow(): void {
        this.store.dispatch(new DeleteWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_deleted'));
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
            });
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000;
        if (this.fileTooLarge) {
            return;
        }
        this.iconUpdated = true;
        this._workflow.icon = event.content;
    }
}
