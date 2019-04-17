import {
    Component,
    EventEmitter,
    Input,
    Output
} from '@angular/core';
import {PermissionValue} from 'app/model/permission.model';
import {
    WNode,
    Workflow,
} from 'app/model/workflow.model';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {IPopup} from 'ng2-semantic-ui';

@Component({

    selector: 'app-workflow-menu-wnode-edit',
    templateUrl: './menu.edit.node.html',
    styleUrls: ['./menu.edit.node.scss'],
})
@AutoUnsubscribe()
export class WorkflowWNodeMenuEditComponent {

    // Project that contains the workflow
    @Input() workflow: Workflow;
    @Input() node: WNode;
    @Input() popup: IPopup;
    @Output() event = new EventEmitter<string>();
    permissionEnum = PermissionValue;

    constructor() {

    }

    /*

    rename(): void {
        if (!this.canEdit()) {
            return;
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let node = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        if (!node) {
            return;
        }
        node.name = this.node.name;
        node.ref = this.node.name;
        // Update join
        if (clonedWorkflow.workflow_data.joins) {
            clonedWorkflow.workflow_data.joins.forEach(j => {
                for (let i = 0; i < j.parents.length; i++) {
                    if (j.parents[i].parent_id === node.id) {
                        j.parents[i].parent_name = node.name;
                        break;
                    }
                }
            });
        }

        this.updateWorkflow(clonedWorkflow, null);
    }

     */

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
    }
}
