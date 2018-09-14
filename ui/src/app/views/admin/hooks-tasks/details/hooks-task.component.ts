import { formatDate } from '@angular/common';
import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { HookStatus, TaskExecution, WorkflowHookTask } from '../../../../model/workflow.hook.model';
import { HookService } from '../../../../service/services.module';
import { Column, HTML } from '../../../../shared/table/sorted-table.component';

@Component({
    selector: 'app-hooks-task',
    templateUrl: './hooks-task.html',
    styleUrls: ['./hooks-task.scss']
})
export class HooksTaskComponent {
    columns: Array<Column>;
    task: WorkflowHookTask;
    executions: Array<TaskExecution>;
    loading: boolean;

    constructor(
        private _hookService: HookService,
        private _translate: TranslateService,
        private _route: ActivatedRoute
    ) {
        this.columns = [
            <Column>{
                type: HTML,
                name: '',
                selector: d => {
                    if (d.status === HookStatus.DONE) {
                        return '<i class="check green icon"></i>';
                    } else if (d.status === HookStatus.FAIL) {
                        return '<i class="ban red icon"></i>';
                    } else {
                        return '<i class="wait blue icon"></i>';
                    }
                }
            },
            <Column>{
                name: 'uuid',
                selector: d => d.uuid
            },
            <Column>{
                name: 'created at',
                selector: d => formatDate(new Date(d.timestamp / 1000000), 'short', this._translate.currentLang)
            },
            <Column>{
                name: 'proceed at',
                selector: d => {
                    return d.processing_timestamp ?
                        formatDate(new Date(d.processing_timestamp / 1000000), 'short', this._translate.currentLang) : '-';
                }
            }
        ];
        this._route.params.subscribe(params => {
            const id = params['id'];
            this.loading = true;
            this._hookService.getAdminTaskExecution(id).subscribe(t => {
                this.loading = false;
                this.task = t;
                this.executions = t.executions.map(exec => {
                    if (exec.nb_errors > 0) {
                        exec.status = HookStatus.FAIL;
                    }
                    return exec;
                });
            });
        });
    }
}
