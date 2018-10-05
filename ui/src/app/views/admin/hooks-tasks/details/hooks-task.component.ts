import { formatDate } from '@angular/common';
import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { HookStatus, TaskExecution, WorkflowHookTask } from '../../../../model/workflow.hook.model';
import { HookService } from '../../../../service/services.module';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-hooks-task',
    templateUrl: './hooks-task.html',
    styleUrls: ['./hooks-task.scss']
})
export class HooksTaskComponent {
    codeMirrorConfig: any;
    columns: Array<Column>;
    task: WorkflowHookTask;
    executions: Array<TaskExecution>;
    selectedExecution: TaskExecution;
    selectedExecutionBody: string;
    loading: boolean;

    constructor(
        private _hookService: HookService,
        private _translate: TranslateService,
        private _route: ActivatedRoute
    ) {
        this.codeMirrorConfig = this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: true
        };

        this.columns = [
            <Column>{
                type: ColumnType.ICON,
                selector: d => {
                    if (d.status === HookStatus.DONE) {
                        return ['check', 'green', 'icon'];
                    } else if (d.status === HookStatus.FAIL) {
                        return ['ban', 'red', 'icon'];
                    } else {
                        return ['wait', 'blue', 'icon'];
                    }
                }
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
            },
            <Column>{
                type: ColumnType.LINK,
                name: 'action',
                selector: d => {
                    return {
                        callback: this.selectExecution(d),
                        value: 'open'
                    };
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

    selectExecution(e: TaskExecution) {
        return _ => {
            this.selectedExecution = e
            this.selectedExecutionBody = null;
            if (e.webhook) {
                this.selectedExecutionBody = this.decodeBody(e.webhook.request_body);
            } else if (e.rabbitmq) {
                this.selectedExecutionBody = this.decodeBody(e.rabbitmq.message);
            } else if (e.kafka) {
                this.selectedExecutionBody = this.decodeBody(e.kafka.message);
            }
        };
    }

    decodeBody(v: string): string {
        if (!v) {
            return '';
        }

        const body = atob(v);
        try {
            return JSON.stringify(JSON.parse(body), null, 4);
        } catch (e) {
            return body;
        }
    }
}
