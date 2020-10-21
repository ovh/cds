import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import { SuiActiveModal } from '@richardlt/ng2-semantic-ui';
import { WNode, WNodeHook, Workflow, WorkflowNodeHookConfigValue } from 'app/model/workflow.model';
import { WorkflowNodeRunHookEvent, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DeleteModalComponent } from 'app/shared/modal/delete/delete.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { WorkflowNodeHookDetailsComponent } from 'app/shared/workflow/node/hook/details/hook.details.component';
import { ProjectState } from 'app/store/project.state';
import { DeleteHookWorkflow, OpenEditModal } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable } from 'rxjs';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeHookComponent implements OnInit, OnDestroy {

    @Input() hook: WNodeHook;
    @Input() workflow: Workflow;
    @Input() node: WNode;

    @Select(WorkflowState.getSelectedWorkflowRun()) workflowRun$: Observable<WorkflowRun>;
    workflowRunSub: Subscription;

    @Select(WorkflowState.getWorkflow()) workflow$: Observable<Workflow>;
    workflowSub: Subscription;

    @ViewChild('deleteHookModal')
    deleteHookModal: DeleteModalComponent;
    @ViewChild('workflowDetailsHook')
    workflowDetailsHook: WorkflowNodeHookDetailsComponent;

    projectKey: string;
    hookEvent: WorkflowNodeRunHookEvent;
    currentRunID: number;
    isReadOnly: boolean;
    icon: string;

    constructor(
        private _store: Store,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef
    ) {
        this.projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflow = this._store.selectSnapshot(WorkflowState.workflowSnapshot);
        this.isReadOnly = !workflow.permissions.writable || (!!workflow.from_template && !!workflow.from_repository);
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        // Check if hook event has changed
        this.workflowRunSub = this.workflowRun$.subscribe(wr => {
            if (!wr) {
                return;
            }
            if (wr.id === this.currentRunID) {
                return;
            }
            if (wr && this.node && wr.nodes && wr.nodes[this.node.id] && wr.nodes[this.node.id].length > 0) {
                let nodeRun = wr.nodes[this.node.id][0];
                this.hookEvent = nodeRun.hook_event;
                this.currentRunID = wr.id;
                this.isReadOnly = true;
                this._cd.markForCheck();
            }
        });

        if (this.hook) {
            if (this.hook.config['hookIcon']) {
                this.icon = (<WorkflowNodeHookConfigValue>this.hook.config['hookIcon']).value.toLowerCase();
            } else if (this.workflow.hook_models && this.workflow.hook_models[this.hook.hook_model_id]) {
                this.icon = this.workflow.hook_models[this.hook.hook_model_id].icon.toLowerCase();
            } else {
                this.icon = this.hook.model.icon;
            }
        }
    }

    receivedEvent(e: string): void {
        switch (e) {
            case 'logs':
                // display logs
                this.workflowDetailsHook.show(this.hook);
                break;
            case 'edit':
                this._store.dispatch(new OpenEditModal({
                    node: this.node,
                    hook: this.hook
                }));
                break;
            case 'delete':
                if (this.deleteHookModal) {
                    this.deleteHookModal.show();
                }
                break
        }
    }

    deleteHook(modal: SuiActiveModal<boolean, boolean, void>) {
        let editMode = this._store.selectSnapshot(WorkflowState).editMode;
        this._cd.markForCheck();
        this._store.dispatch(new DeleteHookWorkflow({
            projectKey: this.projectKey,
            workflowName: this.workflow.name,
            hook: this.hook
        })).subscribe(() => {
            if (editMode) {
                this._toast.info('', this._translate.instant('workflow_ascode_updated'));
            } else {
                this._toast.success('', this._translate.instant('workflow_updated'));
            }
            modal.approve(null);
        });
    }
}
