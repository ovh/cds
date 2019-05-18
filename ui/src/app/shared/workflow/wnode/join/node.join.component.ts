import { Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { UpdateWorkflow } from 'app/store/workflows.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { PipelineStatus } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { WNode, WNodeJoin, Workflow } from '../../../../model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from '../../../../model/workflow.run.model';
import { WorkflowCoreService } from '../../../../service/workflow/workflow.core.service';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { ToastService } from '../../../toast/ToastService';

@Component({
    selector: 'app-workflow-wnode-join',
    templateUrl: './node.join.html',
    styleUrls: ['./node.join.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeJoinComponent {
    @Input() public project: Project;
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public workflowrun: WorkflowRun;
    @Input() public selected: boolean;

    pipelineStatus = PipelineStatus;
    linkJoinSubscription: Subscription;
    nodeToLink: WNode;
    loading = false;

    constructor(
        private _workflowCore: WorkflowCoreService,
        private store: Store,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.linkJoinSubscription = _workflowCore.getLinkJoinEvent().subscribe(n => {
            this.nodeToLink = n;
        });
    }

    selectJoinToLink(): void {
        let cloneWorkflow = cloneDeep(this.workflow);
        let currentJoin = Workflow.getNodeByID(this.node.id, cloneWorkflow);
        if (currentJoin.parents.findIndex(p => p.parent_name === this.nodeToLink.ref) === -1) {
            let joinParent = new WNodeJoin();
            joinParent.parent_name = this.nodeToLink.ref;
            currentJoin.parents.push(joinParent);
        }
        this._workflowCore.linkJoinEvent(null);
        this.updateWorkflow(cloneWorkflow);
    }

    updateWorkflow(w: Workflow): void {
        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.project.key,
            workflowName: w.name,
            changes: w
        })).subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }
}
