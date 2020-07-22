import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnDestroy, Output } from '@angular/core';
import { IPopup } from '@richardlt/ng2-semantic-ui';
import { WNodeHook, Workflow } from 'app/model/workflow.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-menu-hook-edit',
    templateUrl: './menu.edit.hook.html',
    styleUrls: ['./menu.edit.hook.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowHookMenuEditComponent implements OnDestroy {

    // Project that contains the workflow
    @Input() workflow: Workflow;
    @Input() hook: WNodeHook;
    @Input() popup: IPopup;
    @Input() readonly = true;
    @Output() event = new EventEmitter<string>();

    constructor() {}

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
    }
}
