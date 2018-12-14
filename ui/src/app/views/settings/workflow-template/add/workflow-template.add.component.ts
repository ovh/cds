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
        let wt = new WorkflowTemplate();
        wt.editable = true;
        this.workflowTemplate = wt;
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

    saveWorkflowTemplate() {
        this.loading = true;
        this._workflowTemplateService.addWorkflowTemplate(this.workflowTemplate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wt => {
                this.workflowTemplate = wt;
                this._toast.success('', this._translate.instant('workflow_template_created'));
                this._router.navigate(['settings', 'workflow-template', this.workflowTemplate.group.name, this.workflowTemplate.slug]);
            });
    }
}
