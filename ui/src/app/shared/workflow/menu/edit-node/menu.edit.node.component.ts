import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnDestroy,
    OnInit,
    Output
} from '@angular/core';
import { Select, Store } from '@ngxs/store';
import { IPopup } from '@richardlt/ng2-semantic-ui';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { map } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-menu-wnode-edit',
    templateUrl: './menu.edit.node.html',
    styleUrls: ['./menu.edit.node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeMenuEditComponent implements OnInit, OnDestroy {

    // Project that contains the workflow
    @Input() popup: IPopup;
    @Output() event = new EventEmitter<string>();

    project: Project;
    workflow: Workflow;
    node: WNode;


    workflowrun: WorkflowRun;
    @Select(WorkflowState.getSelectedWorkflowRun()) workflowRun$: Observable<WorkflowRun>;
    workflowRunSub: Subscription;

    noderun: WorkflowNodeRun;
    nodeRunSub: Subscription;

    runnable: boolean;
    readonly = true;

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {}

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);

        let state: WorkflowStateModel = this._store.selectSnapshot(WorkflowState);
        this.workflow = state.workflow;
        this.node = state.node;


        this.workflowRunSub = this.workflowRun$.subscribe(wr => {
            if (!wr) {
               return;
            }
            if (this.workflow.id !== wr.workflow_id) {
               return;
            }
            this.workflowrun = wr;
            this.runnable = this.getCanBeRun();
            this._cd.markForCheck();
        });

        this.nodeRunSub = this._store.select(WorkflowState.nodeRunByNodeID)
            .pipe(map(filterFn => filterFn(this.node.id))).subscribe( nodeRun => {
                if (!nodeRun) {
                    return;
                }
                if (this.noderun && this.noderun.status === nodeRun.status) {
                    return;
                }
                this.noderun = nodeRun;
                this.runnable = this.getCanBeRun();
                this._cd.markForCheck();
            });


        this.readonly = !state.canEdit || (!!this.workflow.from_template && !!this.workflow.from_repository);
        this.runnable = this.getCanBeRun();
        this._cd.markForCheck();
    }

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
    }

    getCanBeRun(): boolean {
        if (!this.workflow) {
            return;
        }

        if (this.workflow && !this.workflow.permissions.executable) {
            return false;
        }

        if (this.workflowrun && this.workflowrun.read_only) {
            return false;
        }

        // If we are in a run, check if current node can be run ( compute by cds api)
        if (this.noderun && this.workflowrun && this.workflowrun.nodes) {
            let nodesRun = this.workflowrun.nodes[this.noderun.workflow_node_id];
            if (nodesRun) {
                let nodeRun = nodesRun.find(n => n.id === this.noderun.id);
                if (nodeRun) {
                    return nodeRun.can_be_run;
                }
            }
            return false;
        }

        let workflowrunIsNotActive = this.workflowrun && !PipelineStatus.isActive(this.workflowrun.status);
        if (workflowrunIsNotActive && this.noderun) {
            return true;
        }

        if (this.node && this.workflowrun) {
            if (workflowrunIsNotActive && !this.noderun &&
                this.node.id === this.workflowrun.workflow.workflow_data.node.id) {
                return true;
            }

            if (this.workflowrun && this.workflowrun.workflow && this.workflowrun.workflow.workflow_data.node.id > 0) {
                let nbNodeFound = 0;
                let parentNodes = Workflow.getParentNodeIds(this.workflowrun, this.node.id);
                for (let parentNodeId of parentNodes) {
                    for (let nodeRunId in this.workflowrun.nodes) {
                        if (!this.workflowrun.nodes[nodeRunId]) {
                            continue;
                        }
                        let nodeRuns = this.workflowrun.nodes[nodeRunId];
                        if (nodeRuns[0].workflow_node_id === parentNodeId) { // if node id is still the same
                            if (PipelineStatus.isActive(nodeRuns[0].status)) {
                                return false;
                            }
                            nbNodeFound++;
                        } else if (!Workflow.getNodeByID(nodeRuns[0].workflow_node_id, this.workflowrun.workflow)) {
                            // workflow updated so prefer return true
                            return true;
                        }
                    }
                }
                if (nbNodeFound !== parentNodes.length) { // It means that a parent node isn't already executed
                    return false;
                }
            }
        }
        return true;
    }
}
