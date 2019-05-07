import {
    Component,
    EventEmitter,
    Input,
    Output
} from '@angular/core';
import {PermissionValue} from 'app/model/permission.model';
import {
    WNodeHook,
    Workflow,
} from 'app/model/workflow.model';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {IPopup} from 'ng2-semantic-ui';

@Component({
    selector: 'app-workflow-menu-hook-edit',
    templateUrl: './menu.edit.hook.html',
    styleUrls: ['./menu.edit.hook.scss'],
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
