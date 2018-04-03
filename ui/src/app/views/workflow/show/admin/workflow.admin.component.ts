import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';

import { Project } from '../../../../model/project.model';
import { Workflow } from '../../../../model/workflow.model';
import { WorkflowStore } from '../../../../service/workflow/workflow.store';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../shared/toast/ToastService';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {cloneDeep} from 'lodash';
import {finalize, first} from 'rxjs/operators';

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
            if (this._workflow.purge_tags && this._workflow.purge_tags.length) {
              this.purgeTag = this._workflow.purge_tags[0];
            }
        }
    };
    get workflow() { return this._workflow};

    oldName: string;

    runnumber: number;

    existingTags = new Array<string>();
    selectedTags = new Array<string>();
    purgeTag: string;

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
            this.existingTags.push(...this.selectedTags);
        }

        if (this.project.permission !== 7) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }
        this.oldName = this.workflow.name;

        this._workflowRunService.getTags(this.project.key, this._tagWorkflow.name).subscribe(tags => {
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
        this._tagWorkflow.purge_tags = [this.purgeTag];
        this._workflowStore.updateWorkflow(this.project.key, this._tagWorkflow)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }

    updateTagMetadata(m): void {
        this._tagWorkflow.metadata['default_tags'] = m.join(',');
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
