import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from 'ng2-translate';

import { Project } from '../../../../model/project.model';
import { Workflow } from '../../../../model/workflow.model';
import { WorkflowStore } from '../../../../service/workflow/workflow.store';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../shared/toast/ToastService';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {cloneDeep} from 'lodash';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-admin',
    templateUrl: 'workflow.admin.component.html',
    styleUrls: ['./workflow.admin.scss']
})

export class WorkflowAdminComponent implements OnInit {

    @Input() project: Project;

    _tagWorkflow: Workflow;
    _workflow: Workflow;
    @Input('workflow')
    set workflow (data: Workflow) {
        if (data) {
            this._workflow = cloneDeep(data);
            this._tagWorkflow = cloneDeep(data);
        }
    };
    get workflow() { return this._workflow};

    oldName: string;

    existingTags = new Array<string>();
    selectedTags = new Array<string>();

    @ViewChild('updateWarning')
    private warningUpdateModal: WarningModalComponent;

    loading = false;

    constructor(public _translate: TranslateService, private _toast: ToastService, private _workflowStore: WorkflowStore,
        private _router: Router, private _workflowRunService: WorkflowRunService) { }

    ngOnInit(): void {
        if (!this._tagWorkflow.metadata) {
            this._tagWorkflow.metadata = new Map<string, string>();
        }
        if (this._tagWorkflow.metadata['default_tags']) {
            this.selectedTags = this._tagWorkflow.metadata['default_tags'].split(',');
        }

        if (this.project.permission !== 7) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }
        this.oldName = this.workflow.name;

        this._workflowRunService.getTags(this.project.key, this._tagWorkflow.name).subscribe(tags => {
            Object.keys(tags).forEach(k => {
                if (tags.hasOwnProperty(k)) {
                    this.existingTags.push(k)
                }
            });
        });
    }

    updateWorkflow(): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, this._tagWorkflow).pipe(finalize(() => {
            this.loading = false;
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
        });
    }

    updateTagMetadata(m): void {
        this._tagWorkflow.metadata['default_tags'] = m.join(',');
    }

    onSubmitWorkflowUpdate(skip?: boolean) {
        if (!skip && this.workflow.externalChange) {
            this.warningUpdateModal.show();
        } else {
            this.loading = true;
            this._workflowStore.renameWorkflow(this.project.key, this.oldName, this.workflow).pipe(finalize(() => {
                this.loading = false;
            })).subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name], { queryParams: {tab: 'advanced'}});
            });
        }
    };

    deleteWorkflow(): void {
        this._workflowStore.deleteWorkflow(this.project.key, this.workflow).pipe(finalize(() => {
            this.loading = false;
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_deleted'));
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
        });
    }
}
