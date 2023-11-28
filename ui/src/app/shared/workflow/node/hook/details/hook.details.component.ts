import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    inject,
    OnDestroy,
    OnInit,
    ViewChild
} from '@angular/core';
import { Store } from '@ngxs/store';
import { HookStatus, TaskExecution } from 'app/model/workflow.hook.model';
import { WNodeHook, Workflow } from 'app/model/workflow.model';
import { HookService } from 'app/service/hook/hook.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import {NZ_MODAL_DATA, NzModalRef} from 'ng-zorro-antd/modal';

interface IModalData {
    currentHook: WNodeHook
}

@Component({
    selector: 'app-workflow-node-hook-details',
    templateUrl: './hook.details.component.html',
    styleUrls: ['./hook.details.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeHookDetailsComponent implements OnInit, OnDestroy {
    @ViewChild('code') codemirror: any;

    task: TaskExecution;
    executions: Array<TaskExecution>;
    codeMirrorConfig: any;
    themeSubscription: Subscription;
    body: string;
    loading = true;
    runNumber = 0;

    hookStatus = HookStatus;

    readonly nzModalData: IModalData = inject(NZ_MODAL_DATA);

    constructor(
        public _modal: NzModalRef,
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _hookService: HookService
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: true
        };
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        let project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        let workflow: Workflow;
        let run = this._store.selectSnapshot(WorkflowState.workflowRunSnapshot);
        if (run) {
            this.runNumber = run.num;
            workflow = run.workflow;
        } else {
            workflow = this._store.selectSnapshot(WorkflowState.workflowSnapshot);
        }
        this._hookService.getHookLogs(project.key, workflow.name, this.nzModalData.currentHook.uuid)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((hk) => {
                if (Array.isArray(hk.executions) && hk.executions.length) {
                    hk.executions = hk.executions.map((exec) => {
                        if (exec.nb_errors > 0) {
                            exec.status = HookStatus.FAIL;
                        }
                        return exec;
                    });
                    this.executions = hk.executions;
                    this.task = hk.executions.find(t => t.workflow_run === this.runNumber);
                    this.initDiplayTask();
                    this._cd.markForCheck();
                }
            });

    }

    initDiplayTask(): void {
        if (!this.task) {
            return;
        }
        let jsonBody;
        if (this.task.webhook) {
            jsonBody = atob(this.task.webhook.request_body);
        } else if (this.task.gerrit) {
            jsonBody = atob(this.task.gerrit.message);
        }
        try {
            this.body = JSON.stringify(JSON.parse(jsonBody), null, 4);
        } catch (e) {
            this.body = jsonBody;
        }
    }

    selectTask(exec: TaskExecution): void {
        this.task = exec;
        this.initDiplayTask();
        this._cd.markForCheck();
    }
}
