import {Component, Input, OnInit, ViewChild} from '@angular/core';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs/Subscription';
import { Project } from '../../../../model/project.model';
import { WNode, WNodeHook, Workflow, WorkflowNodeHookConfigValue } from '../../../../model/workflow.model';
import { WorkflowEventStore } from '../../../../service/workflow/workflow.event.store';
import {OpenWorkflowNodeModal} from 'app/store/node.modal.action';
import {Store} from '@ngxs/store';
import {DeleteModalComponent} from 'app/shared/modal/delete/delete.component';

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
        private _store: Store
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
            case 'edit':
                this._store.dispatch(new OpenWorkflowNodeModal({
                    project: this.project,
                    workflow: this.workflow,
                    node: null,
                    hook: this.hook
                })).subscribe(() => {});
                break;
            case 'delete':

                break
        }
    }
}
