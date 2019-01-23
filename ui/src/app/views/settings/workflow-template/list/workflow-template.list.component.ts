import { Component } from '@angular/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { WorkflowTemplate } from '../../../../model/workflow-template.model';
import { WorkflowTemplateService } from '../../../../service/workflow-template/workflow-template.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-workflow-template-list',
    templateUrl: './workflow-template.list.html',
    styleUrls: ['./workflow-template.list.scss']
})
export class WorkflowTemplateListComponent {
    loading: boolean;
    columns: Array<Column<WorkflowTemplate>>;
    workflowTemplates: Array<WorkflowTemplate>;

    path: Array<PathItem>

    constructor(
        private _workflowTemplateService: WorkflowTemplateService
    ) {
        this.columns = [
            <Column<WorkflowTemplate>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (wt: WorkflowTemplate) => {
                    return {
                        link: '/settings/workflow-template/' + wt.group.name + '/' + wt.slug,
                        value: wt.name
                    };
                }
            },
            <Column<WorkflowTemplate>>{
                type: ColumnType.MARKDOWN,
                name: 'common_description',
                selector: (wt: WorkflowTemplate) => wt.description
            },
            <Column<WorkflowTemplate>>{
                name: 'common_group',
                selector: (wt: WorkflowTemplate) => wt.group.name
            }
        ];
        this.getTemplates();

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'workflow_templates',
            routerLink: ['/', 'settings', 'workflow-template']
        }];
    }

    getTemplates() {
        this.loading = true;
        this._workflowTemplateService.getAll()
            .pipe(finalize(() => this.loading = false))
            .subscribe(wts => { this.workflowTemplates = wts; });
    }
}
