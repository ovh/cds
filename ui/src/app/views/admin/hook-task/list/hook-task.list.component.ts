import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { WorkflowHookTask } from 'app/model/workflow.hook.model';
import { HookService } from 'app/service/hook/hook.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-hook-task-list',
    templateUrl: './hook-task.list.html',
    styleUrls: ['./hook-task.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class HookTaskListComponent {
    loading = false;
    columns: Array<Column<WorkflowHookTask>>;
    tasks: Array<WorkflowHookTask>;
    filter: Filter<WorkflowHookTask>;
    dataCount: number;
    path: Array<PathItem>;

    constructor(
        private _hookService: HookService,
        private _cd: ChangeDetectorRef
        ) {
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
            <Column<WorkflowHookTask>>{
                type: ColumnType.ICON,
                selector: (d: WorkflowHookTask) => d.stopped ? ['stop', 'red', 'icon'] : ['play', 'green', 'icon']
            },
            <Column<WorkflowHookTask>>{
                type: ColumnType.ROUTER_LINK,
                name: 'UUID',
                selector: (d: WorkflowHookTask) => ({
                        link: '/admin/hooks-tasks/' + d.uuid,
                        value: d.uuid
                    })
            },
            <Column<WorkflowHookTask>>{
                name: 'common_type',
                selector: (d: WorkflowHookTask) => d.type
            },
            <Column<WorkflowHookTask>>{
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
            <Column<WorkflowHookTask>>{
                name: 'hook_task_execs_todo',
                selector: (d: WorkflowHookTask) => d.nb_executions_todo,
                sortable: true,
                sortKey: 'nb_executions_todo'
            },
            <Column<WorkflowHookTask>>{
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
        this._hookService.getAdminTasks(sort).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(ts => {
            this.tasks = ts;

        });
    }

    dataChange(count: number) {
        this.dataCount = count;
    }
}
