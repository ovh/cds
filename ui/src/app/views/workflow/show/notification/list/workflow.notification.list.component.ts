import {Component, Input} from '@angular/core';
import {Workflow} from '../../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-notification-list',
    templateUrl: './workflow.notification.list.html',
    styleUrls: ['./workflow.notification.list.scss']
})
export class WorkflowNotificationListComponent {

    mapNodesNotif: Map<number, Array<string>>;
    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = data;
            this.refreshNotif();
        }
    }

    get workflow() {
        return this._workflow;
    }

    refreshNotif(): void {
        let mapNodes = Workflow.getMapNodes(this.workflow);
        this.mapNodesNotif = new Map<number, Array<string>>();
        if (this.workflow.notifications) {
            this.workflow.notifications.forEach(n => {
                let listNodes = new Array<string>();
                n.source_node_id.forEach(id => {
                    let node = mapNodes.get(id);
                    if (node) {
                        listNodes.push(node.name);
                    }
                });
                this.mapNodesNotif.set(n.id, listNodes);
            });
        }
    }

    constructor() {
    }
}
