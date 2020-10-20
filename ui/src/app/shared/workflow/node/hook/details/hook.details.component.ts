import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { HookStatus, TaskExecution } from 'app/model/workflow.hook.model';
import { WNodeHook, Workflow } from 'app/model/workflow.model';
import { HookService } from 'app/service/hook/hook.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-hook-details',
    templateUrl: './hook.details.component.html',
    styleUrls: ['./hook.details.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeHookDetailsComponent implements OnInit, OnDestroy {
    @ViewChild('code') codemirror: any;
    @ViewChild('nodeHookDetailsModal') nodeHookDetailsModal: ModalTemplate<boolean, boolean, void>;

    modal: SuiActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;
    task: TaskExecution;
    executions: Array<TaskExecution>;
    codeMirrorConfig: any;
    themeSubscription: Subscription;
    body: string;
    loading = true;
    runNumber = 0;

    hookStatus = HookStatus;

    constructor(
        private _modalService: SuiModalService,
        private _theme: ThemeStore,
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

    ngOnDestroy(): void {
    } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.themeSubscription = this._theme.get()
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(t => {
                this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
                if (this.codemirror && this.codemirror.instance) {
                    this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
                }
            });
    }

    show(hook: WNodeHook): void {
        let project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        let workflow: Workflow;
        let run = this._store.selectSnapshot(WorkflowState.workflowRunSnapshot)
        if (run) {
            this.runNumber = run.num;
            workflow = run.workflow;
        } else {
            workflow = this._store.selectSnapshot(WorkflowState.workflowSnapshot);
        }
        this._hookService.getHookLogs(project.key, workflow.name, hook.uuid)
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
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeHookDetailsModal);
        this.modalConfig.size = 'large';
        this.modal = this._modalService.open(this.modalConfig);
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
