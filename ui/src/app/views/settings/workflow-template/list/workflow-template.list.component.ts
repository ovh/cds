import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { WorkflowTemplate } from 'app/model/workflow-template.model';
import { WorkflowTemplateService } from 'app/service/workflow-template/workflow-template.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-template-list',
    templateUrl: './workflow-template.list.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTemplateListComponent {
    loading: boolean;
    columns: Array<Column<WorkflowTemplate>>;
    workflowTemplates: Array<WorkflowTemplate>;
    path: Array<PathItem>;
    filter: Filter<WorkflowTemplate>;

    constructor(private _workflowTemplateService: WorkflowTemplateService, private _cd: ChangeDetectorRef) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                let s = `${d.group.name}/${d.name}`.toLowerCase();
                return s.indexOf(lowerFilter) !== -1 || d.description.toLowerCase().indexOf(lowerFilter) !== -1;
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
                name: 'common_group',
                class: 'three',
                selector: (wt: WorkflowTemplate) => wt.group.name
            },
            <Column<WorkflowTemplate>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                class: 'four',
                selector: (wt: WorkflowTemplate) => ({
                        link: `/settings/workflow-template/${wt.group.name}/${wt.slug}`,
                        value: wt.name
                    })
            },
            <Column<WorkflowTemplate>>{
                type: ColumnType.MARKDOWN,
                name: 'common_description',
                selector: (wt: WorkflowTemplate) => {
                    if (wt.description && wt.description.length > 100) {
                        return wt.description.substr(0, 100) + '...';
                    }
                    return wt.description;
                }
            }
        ];

        this.getTemplates();
    }

    getTemplates() {
        this.loading = true;
        this._workflowTemplateService.getAll()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wts => {
                this.workflowTemplates = wts.sort((a, b) => {
                    let aG = a.group.name.toLowerCase();
                    let bG = b.group.name.toLowerCase();
                    if (aG === bG) {
                        return a.name.toLowerCase() > b.name.toLowerCase() ? 1 : -1;
                    }
                    return aG > bG ? 1 : -1;
                });
            });
    }
}
