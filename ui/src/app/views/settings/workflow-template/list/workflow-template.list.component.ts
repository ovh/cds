import { Component } from '@angular/core';
import { WorkflowTemplate } from 'app/model/workflow-template.model';
import { WorkflowTemplateService } from 'app/service/workflow-template/workflow-template.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/internal/operators/finalize';

@Component({
    selector: 'app-workflow-template-list',
    templateUrl: './workflow-template.list.html'
})
export class WorkflowTemplateListComponent {
    loading: boolean;
    columns: Array<Column<WorkflowTemplate>>;
    workflowTemplates: Array<WorkflowTemplate>;
    path: Array<PathItem>;
    filter: Filter<WorkflowTemplate>;

    constructor(
        private _workflowTemplateService: WorkflowTemplateService
    ) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                let s = `${d.group.name}/${d.name}`.toLowerCase();
                return s.indexOf(lowerFilter) !== -1;
            }
        };

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'workflow_templates',
            routerLink: ['/', 'settings', 'workflow-template']
        }];

        this.columns = [
            <Column<WorkflowTemplate>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (wt: WorkflowTemplate) => {
                    return {
                        link: `/settings/workflow-template/${wt.group.name}/${wt.slug}`,
                        value: wt.name
                    };
                }
            },
            <Column<WorkflowTemplate>>{
                name: 'common_group',
                selector: (wt: WorkflowTemplate) => wt.group.name
            },
            <Column<WorkflowTemplate>>{
                type: ColumnType.MARKDOWN,
                name: 'common_description',
                selector: (wt: WorkflowTemplate) => wt.description
            }
        ];

        this.getTemplates();
    }

    getTemplates() {
        this.loading = true;
        this._workflowTemplateService.getAll()
            .pipe(finalize(() => this.loading = false))
            .subscribe(wts => { this.workflowTemplates = wts; });
    }
}
