import { Component } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { WorkflowTemplateService } from '../../../../service/workflow-template/workflow-template.service';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-workflow-template-list',
    templateUrl: './workflow-template.list.html',
    styleUrls: ['./workflow-template.list.scss']
})
export class WorkflowTemplateListComponent {
    loading: boolean;
    columns: Array<Column>;
    workflowTemplates: Array<WorkflowTemplate>;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService,
        private _translate: TranslateService
    ) {
        this.columns = [
            <Column>{
                type: ColumnType.ROUTER_LINK,
                name: this._translate.instant('common_name'),
                selector: wt => {
                    return {
                        link: '/settings/workflow-template/' + wt.id,
                        value: wt.name
                    };
                }
            },
            <Column>{
                name: this._translate.instant('common_description'),
                selector: wt => wt.description
            },
            <Column>{
                name: this._translate.instant('common_group'),
                selector: wt => wt.group.name
            }
        ];
        this.getTemplates();
    }

    getTemplates() {
        this.loading = true;
        this._workflowTemplateService.getWorkflowTemplates()
            .pipe(finalize(() => this.loading = false))
            .subscribe(wts => { this.workflowTemplates = wts; });
    }
}
