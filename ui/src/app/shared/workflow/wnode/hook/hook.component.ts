import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Store} from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { WNode, WNodeHook, Workflow, WorkflowNodeHookConfigValue } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { WorkflowEventStore } from 'app/service/workflow/workflow.event.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {DeleteModalComponent} from 'app/shared/modal/delete/delete.component';
import {ToastService} from 'app/shared/toast/ToastService';
import {OpenWorkflowNodeModal} from 'app/store/node.modal.action';
import {DeleteHookWorkflow} from 'app/store/workflows.action';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {finalize} from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
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

    @ViewChild('deleteHookModal')
    deleteHookModal: DeleteModalComponent;

    icon: string;
    loading = false;
    isSelected = false;
    subSelect: Subscription;
    subRun: Subscription;
    nodeRun: WorkflowNodeRun;

    constructor(
        private _workflowEventStore: WorkflowEventStore,
        private _store: Store, private _toast: ToastService, private _translate: TranslateService
    ) { }

    ngOnInit(): void {
        this.subSelect = this._workflowEventStore.selectedHook().subscribe(h => {
            if (this.hook && h) {
                this.isSelected = h.uuid === this.hook.uuid;
                return;
            }
            this.isSelected = false;
        });

        if (this._hook) {
            if (this._hook.config['hookIcon']) {
                this.icon = (<WorkflowNodeHookConfigValue>this._hook.config['hookIcon']).value.toLowerCase();
            } else {
                this.icon = this.workflow.hook_models[this.hook.hook_model_id].icon.toLowerCase();
            }
        }

        // Get workflow run
        this.subRun = this._workflowEventStore.selectedRun().subscribe(wr => {
            this.workflowRun = wr;
            if (wr && wr.nodes && this.node && wr.nodes[this.node.id] && wr.nodes[this.node.id].length > 0) {
                this.nodeRun = this.workflowRun.nodes[this.node.id][0];
            } else {
                this.nodeRun = null;
            }
        });
    }

    receivedEvent(e: string): void {
        switch (e) {
            case 'details':
                this._workflowEventStore.setSelectedHook(this.hook);
                 break;
            case 'edit':
                this._store.dispatch(new OpenWorkflowNodeModal({
                    project: this.project,
                    workflow: this.workflow,
                    node: this.node,
                    hook: this.hook
                })).subscribe(() => {});
                break;
            case 'delete':
                if (this.deleteHookModal) {
                    this.deleteHookModal.show();
                }
                break
        }
    }

    deleteHook(modal: ActiveModal<boolean, boolean, void>) {
        this.loading = true;
        this._store.dispatch(new DeleteHookWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            hook: this.hook
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                this._workflowEventStore.unselectAll();
                modal.approve(null);
            });
    }
}
