import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {Subscription} from 'rxjs';
import {PermissionValue} from '../../../../model/permission.model';
import {Project} from '../../../../model/project.model';
import {WNode, Workflow} from '../../../../model/workflow.model';
import {WorkflowEventStore} from '../../../../service/services.module';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {WorkflowDeleteNodeComponent} from '../../modal/delete/workflow.node.delete.component';

@Component({
    selector: 'app-workflow-sidebar-wnode-edit',
    templateUrl: './sidebar.edit.html',
    styleUrls: ['./sidebar.edit.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeSidebarEditComponent implements OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    node: WNode;
    nodeSub: Subscription;

    displayInputName = false;
    previousNodeName: string;

    permissionEnum = PermissionValue;
    loading = false;

    @ViewChild('workflowDeleteNode')
    workflowDeleteNode: WorkflowDeleteNodeComponent;

    constructor(private _workflowEventStore: WorkflowEventStore) {

    }

    ngOnInit(): void {
        this.nodeSub = this._workflowEventStore.selectedNode().subscribe(n => {
            if (n) {
                if (!this.displayInputName) {
                    this.previousNodeName = n.name
                }
            }
            this.node = n;
        });
    }

    canEdit(): boolean {
        return this.workflow.permission === PermissionValue.READ_WRITE_EXECUTE;
    }

    openDeleteNodeModal(): void {
        if (!this.canEdit()) {
            return;
        }
        if (this.workflowDeleteNode) {
            this.workflowDeleteNode.show();
        }
    }

    updateWorkflow(w: Workflow): void {
        console.log(w);
    }
}
