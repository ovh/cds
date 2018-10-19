import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { Group } from '../../../../model/group.model';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { GroupService } from '../../../../service/services.module';
import { WorkflowTemplateService } from '../../../../service/workflow-template/workflow-template.service';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-template-add',
    templateUrl: './workflow-template.add.html',
    styleUrls: ['./workflow-template.add.scss']
})
export class WorkflowTemplateAddComponent {
    workflowTemplate: WorkflowTemplate;
    groups: Array<Group>;
    loading: boolean;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _groupService: GroupService,
        private _router: Router,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.getGroups();
    }

    getGroups() {
        this.loading = true;
        this._groupService.getGroups()
            .pipe(finalize(() => this.loading = false))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    saveWorkflowTemplate(wt: WorkflowTemplate) {
        this.loading = true;
        this._workflowTemplateService.addWorkflowTemplate(wt)
            .pipe(finalize(() => this.loading = false))
            .subscribe(res => {
                this.workflowTemplate = res;
                this._toast.success('', this._translate.instant('workflow_template_created'));
                this._router.navigate(['settings', 'workflow-template', this.workflowTemplate.id]);
            });
    }
}
