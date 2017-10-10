import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from 'ng2-translate';

import { Project } from '../../../../model/project.model';
import { Workflow } from '../../../../model/workflow.model';
import { WorkflowStore } from '../../../../service/workflow/workflow.store';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-admin',
    templateUrl: 'workflow.admin.component.html',
    styleUrls: ['./workflow.admin.scss']
})

export class WorkflowAdminComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;

    oldName: string;

    @ViewChild('updateWarning')
    private warningUpdateModal: WarningModalComponent;

    loading = false;

    constructor(public _translate: TranslateService, private _toast: ToastService, private _workflowStore: WorkflowStore,
        private _router: Router) { }

    ngOnInit(): void {
        if (this.project.permission !== 7) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }
        this.oldName = this.workflow.name;
    }

    onSubmitWorkflowUpdate(skip?: boolean) {
        if (!skip && this.workflow.externalChange) {
            this.warningUpdateModal.show();
        } else {
            this.loading = true;
            this._workflowStore.renameWorkflow(this.project.key, this.oldName, this.workflow).finally(() => {
                this.loading = false;
            }).subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
        }
    };

    deleteWorkflow(): void {
        this._workflowStore.deleteWorkflow(this.project.key, this.workflow).finally(() => {
            this.loading = false;
        }).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_deleted'));
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
        });
    }
}
