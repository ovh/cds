import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnDestroy, OnInit, Output } from '@angular/core';
import { Store } from '@ngxs/store';
import { IPopup } from '@richardlt/ng2-semantic-ui';
import { WNodeHook, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRunHookEvent } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowState } from 'app/store/workflow.state';

@Component({
    selector: 'app-workflow-menu-hook-edit',
    templateUrl: './menu.edit.hook.html',
    styleUrls: ['./menu.edit.hook.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowHookMenuEditComponent implements OnInit, OnDestroy {

    // Project that contains the workflow
    @Input() workflow: Workflow;
    @Input() hook: WNodeHook;
    @Input() popup: IPopup;
    @Input() readonly = true;
    @Input() hookEventUUID: string;
    @Output() event = new EventEmitter<string>();

    hookEvent: WorkflowNodeRunHookEvent;
    isRun: boolean;

    constructor(private _store: Store, private _cd: ChangeDetectorRef) {}

    ngOnInit(): void {
        let wr = this._store.selectSnapshot(WorkflowState.workflowRunSnapshot);
        if (wr) {
            this.isRun = true;
            let rootNodeRun = wr.nodes[wr.workflow.workflow_data.node.id][0]
            if (rootNodeRun && rootNodeRun?.hook_event ) {
                this.hookEvent = rootNodeRun?.hook_event;
            }
            this._cd.markForCheck();
        }
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
    }
}
