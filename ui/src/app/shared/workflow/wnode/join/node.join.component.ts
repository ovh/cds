import { Component, Input } from '@angular/core';
import {PipelineStatus} from '@cds/model/pipeline.model';
import {Project} from '@cds/model/project.model';
import {WNode, WNodeJoin, Workflow} from '@cds/model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '@cds/model/workflow.run.model';
import {WorkflowCoreService} from '@cds/service/workflow/workflow.core.service';
import {AutoUnsubscribe} from '@cds/shared/decorator/autoUnsubscribe';
import {ToastService} from '@cds/shared/toast/ToastService';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { cloneDeep } from 'lodash';
import { Subscription } from 'rxjs';

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
