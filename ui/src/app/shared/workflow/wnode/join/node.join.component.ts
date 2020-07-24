import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, WNodeJoin, Workflow } from 'app/model/workflow.model';
import { WorkflowCoreService } from 'app/service/workflow/workflow.core.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-wnode-join',
    templateUrl: './node.join.html',
    styleUrls: ['./node.join.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeJoinComponent implements OnDestroy {
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() noderunStatus: string;
    @Input() public editMode: boolean;

    project: Project;
    pipelineStatus = PipelineStatus;
    linkJoinSubscription: Subscription;
    nodeToLink: WNode;
    loading = false;

    constructor(
        private _workflowCore: WorkflowCoreService,
        private store: Store,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this.store.selectSnapshot(ProjectState.projectSnapshot);
        this.linkJoinSubscription = _workflowCore.getLinkJoinEvent().subscribe(n => {
            this.nodeToLink = n;
            this._cd.markForCheck();
        });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    selectJoinToLink(): void {
        let cloneWorkflow = cloneDeep(this.workflow);
        let currentJoin: WNode;
        if (this.store.selectSnapshot(WorkflowState).editMode) {
            currentJoin = Workflow.getNodeByRef(this.node.ref, cloneWorkflow);
        } else {
            currentJoin = Workflow.getNodeByID(this.node.id, cloneWorkflow);
        }
        if (currentJoin.parents.findIndex(p => p.parent_name === this.nodeToLink.ref) === -1) {
            let joinParent = new WNodeJoin();
            joinParent.parent_name = this.nodeToLink.ref;
            currentJoin.parents.push(joinParent);
        }
        this._workflowCore.linkJoinEvent(null);
        this.updateWorkflow(cloneWorkflow);
    }

    updateWorkflow(w: Workflow): void {
        let editMode = this.store.selectSnapshot(WorkflowState).editMode;
        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.project.key,
            workflowName: w.name,
            changes: w
        })).subscribe(() => {
            if (!editMode) {
                this._toast.success('', this._translate.instant('workflow_updated'));
            } else {
                this._toast.info('', this._translate.instant('workflow_ascode_updated'));
            }
        });
    }
}
