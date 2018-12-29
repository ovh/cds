import { Component } from '@angular/core';
import { WorkflowHookTask } from '../../../../model/workflow.hook.model';
import { HookService } from '../../../../service/services.module';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-hook-task-list',
    templateUrl: './hook-task.list.html',
    styleUrls: ['./hook-task.list.scss']
})
export class HookTaskListComponent {
    loading = false;
    columns: Array<Column>;
    tasks: Array<WorkflowHookTask>;
    filter: Filter;
    dataCount: number;
    path: Array<PathItem>;

    constructor(private _hookService: HookService) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                let hookPath: string;
                if (d.config && d.config['project'] && d.config['workflow']) {
                    hookPath = (d.config['project'].value + '/' + d.config['workflow'].value).toLowerCase()
                }
                return d.uuid.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.type.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    (hookPath && hookPath.indexOf(lowerFilter) !== -1) ||
                    d.nb_executions_todo.toString().toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.nb_executions_total.toString().toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };

        this.columns = [
            <Column>{
                type: ColumnType.ICON,
                selector: (d: WorkflowHookTask) => d.stopped ? ['stop', 'red', 'icon'] : ['play', 'green', 'icon']
            },
            <Column>{
                type: ColumnType.ROUTER_LINK,
                name: 'UUID',
                selector: (d: WorkflowHookTask) => {
                    return {
                        link: '/admin/hooks-tasks/' + d.uuid,
                        value: d.uuid
                    };
                }
            },
            <Column>{
                name: 'common_type',
                selector: (d: WorkflowHookTask) => d.type
            },
            <Column>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_workflow',
                selector: (d: WorkflowHookTask) => {
                    if (!d.config || !d.config['project'] || !d.config['workflow']) {
                        return {
                            link: '',
                            value: ''
                        }
                    }
                    return {
                        link: '/project/' + d.config['project'].value + '/workflow/' + d.config['workflow'].value,
                        value: d.config['project'].value + '/' + d.config['workflow'].value
                    };
                },
            },
            <Column>{
                name: 'hook_task_execs_todo',
                selector: (d: WorkflowHookTask) => d.nb_executions_todo,
                sortable: true,
                sortKey: 'nb_executions_todo'
            },
            <Column>{
                name: 'hook_task_execs_total',
                selector: (d: WorkflowHookTask) => d.nb_executions_total,
                sortable: true,
                sortKey: 'nb_executions_total'
            }
        ];

        this.getAdminTasks('');

        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'hook_tasks_summary',
            routerLink: ['/', 'admin', 'hooks-tasks']
        }];
    }

    getAdminTasks(sort: string) {
        this.loading = true;
        this._hookService.getAdminTasks(sort).subscribe(ts => {
            this.tasks = ts;
            this.loading = false;
        });
    }

    dataChange(count: number) {
        this.dataCount = count;
    }
}
