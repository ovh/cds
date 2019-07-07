import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { IPopup } from '@richardlt/ng2-semantic-ui';
import { PermissionValue } from 'app/model/permission.model';
import { WNodeHook, Workflow } from 'app/model/workflow.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-menu-hook-edit',
    templateUrl: './menu.edit.hook.html',
    styleUrls: ['./menu.edit.hook.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowHookMenuEditComponent {

    // Project that contains the workflow
    @Input() workflow: Workflow;
    @Input() hook: WNodeHook;
    @Input() popup: IPopup;
    @Input() readonly = true;
    @Output() event = new EventEmitter<string>();
    permissionEnum = PermissionValue;

    constructor() {}

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
    }
}
