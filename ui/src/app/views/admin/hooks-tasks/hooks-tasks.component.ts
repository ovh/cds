import { Component } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { WorkflowHookTask } from '../../../model/workflow.hook.model';
import { HookService } from '../../../service/services.module';
import { Column } from '../../../shared/table/sorted-table.component';

@Component({
    selector: 'app-hooks-tasks',
    templateUrl: './hooks-tasks.html',
    styleUrls: ['./hooks-tasks.scss']
})
export class HooksTasksComponent {
    loading = false;
    columns: Array<Column>;
    tasks: Array<WorkflowHookTask>;

    constructor(
        private _hookService: HookService,
        private _translate: TranslateService
    ) {
        this.loading = true;
        this.columns = [
            {
                name: this._translate.instant('hook_task_cron'),
                selector: d => d.config['cron'] && d.config['cron'].value,
            },
            {
                name: this._translate.instant('hook_task_execs_todo'),
                selector: d => d.nb_executions_todo,
                sortable: true,
                sortKey: 'nb_executions_todo'
            },
            {
                name: this._translate.instant('hook_task_execs_total'),
                selector: d => d.nb_executions_total,
                sortable: true,
                sortKey: 'nb_executions_total'
            },
            {
                name: this._translate.instant('common_project'),
                selector: d => d.config['project'] && d.config['project'].value,
            },
            {
                name: this._translate.instant('hook_task_repo_fullname'),
                selector: d => d.config['repoFullName'] && d.config['repoFullName'].value,
            },
            {
                name: this._translate.instant('common_stopped'),
                selector: d => d.stopped,
            },
            {
                name: this._translate.instant('common_type'),
                selector: d => d.type,
            },
            {
                name: 'UUID',
                selector: d => d.uuid,
            },
            {
                name: this._translate.instant('vcs_server'),
                selector: d => d.config['vcsServer'] && d.config['vcsServer'].value,
            },
            {
                name: this._translate.instant('common_workflow'),
                selector: d => d.config['workflow'] && d.config['workflow'].value,
            }
        ];
        this._hookService.getAdminTasks('')
            .subscribe(ts => {
                this.tasks = ts;
                this.loading = false;
            });
    }

    sortChange(event: string) {
        this._hookService.getAdminTasks(event)
            .subscribe(ts => {
                this.tasks = ts;
                this.loading = false;
            });
    }
}
