import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Store} from '@ngxs/store';
import { SuiActiveModal } from '@richardlt/ng2-semantic-ui';
import { Project } from 'app/model/project.model';
import { WNode, WNodeHook, Workflow, WorkflowNodeHookConfigValue } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {DeleteModalComponent} from 'app/shared/modal/delete/delete.component';
import {ToastService} from 'app/shared/toast/ToastService';
import {DeleteHookWorkflow, OpenEditModal, SelectHook} from 'app/store/workflow.action';
import {WorkflowState, WorkflowStateModel} from 'app/store/workflow.state';
import {finalize} from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeHookComponent implements OnInit {
    _hook: WNodeHook;
    @Input('hook')
    set hook(data: WNodeHook) {
        if (data) {
            this._hook = data;
        }
    }
    get hook() { return this._hook; }
    @Input() readonly = false;
    @Input() workflow: Workflow;
    @Input() workflowRun: WorkflowRun;
    @Input() project: Project;
    @Input() node: WNode;

    @ViewChild('deleteHookModal', {static: false})
    deleteHookModal: DeleteModalComponent;

    icon: string;
    loading = false;
    isSelected = false;
    subSelect: Subscription;
    subRun: Subscription;
    nodeRun: WorkflowNodeRun;

    constructor(
        private _store: Store, private _toast: ToastService, private _translate: TranslateService,
        private _cd: ChangeDetectorRef
    ) {
    }

    ngOnInit(): void {
        this.subSelect = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            this.readonly = !s.canEdit;
            this.workflowRun = s.workflowRun;
            if (this.workflowRun && this.node && this.workflowRun.nodes
                && this.workflowRun.nodes[this.node.id] && this.workflowRun.nodes[this.node.id].length > 0) {
                this.nodeRun = this.workflowRun.nodes[this.node.id][0];
            }

            if (s.hook && this.hook && s.hook.uuid === this.hook.uuid) {
                this.isSelected = true;
            } else {
                this.isSelected = false;
            }
            this._cd.markForCheck();
        });

        if (this._hook) {
            if (this._hook.config['hookIcon']) {
                this.icon = (<WorkflowNodeHookConfigValue>this._hook.config['hookIcon']).value.toLowerCase();
            } else {
                this.icon = this.workflow.hook_models[this.hook.hook_model_id].icon.toLowerCase();
            }
        }
    }

    receivedEvent(e: string): void {
        switch (e) {
            case 'details':
                this._store.dispatch(new SelectHook({hook: this.hook, node: this.node}));
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
        this.loading = true;
        this._store.dispatch(new DeleteHookWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            hook: this.hook
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                modal.approve(null);
            });
    }
}
