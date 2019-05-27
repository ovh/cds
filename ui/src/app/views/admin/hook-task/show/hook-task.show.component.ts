import { formatDate } from '@angular/common';
import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { HookStatus, TaskExecution, WorkflowHookTask } from '../../../../model/workflow.hook.model';
import { HookService } from '../../../../service/services.module';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-hook-task-show',
    templateUrl: './hook-task.show.html',
    styleUrls: ['./hook-task.show.scss']
})
export class HookTaskShowComponent {
    codeMirrorConfig: any;
    columns: Array<Column<TaskExecution>>;
    task: WorkflowHookTask;
    executions: Array<TaskExecution>;
    selectedExecution: TaskExecution;
    selectedExecutionBody: string;
    loading: boolean;
    path: Array<PathItem>;

    constructor(
        private _hookService: HookService,
        private _translate: TranslateService,
        private _route: ActivatedRoute
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: true
        };

        this.columns = [
            <Column<TaskExecution>>{
                type: ColumnType.ICON,
                selector: (d: TaskExecution) => {
                    if (d.status === HookStatus.DONE) {
                        return ['check', 'green', 'icon'];
                    } else if (d.status === HookStatus.FAIL) {
                        return ['ban', 'red', 'icon'];
                    } else {
                        return ['wait', 'blue', 'icon'];
                    }
                }
            },
            <Column<TaskExecution>>{
                name: 'created at',
                selector: (d: TaskExecution) => formatDate(new Date(d.timestamp / 1000000), 'short', this._translate.currentLang)
            },
            <Column<TaskExecution>>{
                name: 'proceed at',
                selector: (d: TaskExecution) => {
                    return d.processing_timestamp ?
                        formatDate(new Date(d.processing_timestamp / 1000000), 'short', this._translate.currentLang) : '-';
                }
            },
            <Column<TaskExecution>>{
                type: ColumnType.LINK,
                name: 'action',
                selector: (d: TaskExecution) => {
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
                this.updatePath();
            });
        });
    }

    selectExecution(e: TaskExecution) {
        return () => {
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

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'hook_tasks_summary',
            routerLink: ['/', 'admin', 'hooks-tasks']
        }];

        if (this.task) {
            this.path.push(<PathItem>{
                text: this.task.uuid,
                routerLink: ['/', 'admin', 'hooks-tasks', this.task.uuid]
            });
        }
    }
}
