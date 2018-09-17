import { Component } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { WorkflowHookTask } from '../../../model/workflow.hook.model';
import { HookService } from '../../../service/services.module';
import { Column, ColumnType, Filter } from '../../../shared/table/sorted-table.component';

@Component({
    selector: 'app-hooks-tasks',
    templateUrl: './hooks-tasks.html',
    styleUrls: ['./hooks-tasks.scss']
})
export class HooksTasksComponent {
    loading = false;
    columns: Array<Column>;
    tasks: Array<WorkflowHookTask>;
    filter: Filter;

    constructor(
        private _hookService: HookService,
        private _translate: TranslateService
    ) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                return d.uuid.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.type.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.config['project'].value.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.config['workflow'].value.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.nb_executions_todo.toString().toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.nb_executions_total.toString().toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };
        this.columns = [
            <Column>{
                type: ColumnType.ICON,
                selector: d => d.stopped ? ['stop', 'red', 'icon'] : ['play', 'green', 'icon']
            },
            <Column>{
                type: ColumnType.ROUTER_LINK,
                name: 'UUID',
                selector: d => {
                    return {
                        link: '/admin/hooks-tasks/' + d.uuid,
                        value: d.uuid
                    };
                }
            },
            <Column>{
                name: this._translate.instant('common_type'),
                selector: d => d.type
            },
            <Column>{
                type: ColumnType.ROUTER_LINK,
                name: this._translate.instant('common_project') + '/' + this._translate.instant('common_workflow'),
                selector: d => {
                    return {
                        link: '/project/' + d.config['project'].value + '/workflow/' + d.config['workflow'].value,
                        value: d.config['project'].value + '/' + d.config['workflow'].value
                    };
                },
            },
            <Column>{
                name: this._translate.instant('hook_task_execs_todo'),
                selector: d => d.nb_executions_todo,
                sortable: true,
                sortKey: 'nb_executions_todo'
            },
            <Column>{
                name: this._translate.instant('hook_task_execs_total'),
                selector: d => d.nb_executions_total,
                sortable: true,
                sortKey: 'nb_executions_total'
            }
        ];
        this.getAdminTasks('');
    }

    getAdminTasks(sort: string) {
        this.loading = true;
        this._hookService.getAdminTasks(sort).subscribe(ts => {
            this.tasks = ts;
            this.loading = false;
        });
    }
}
