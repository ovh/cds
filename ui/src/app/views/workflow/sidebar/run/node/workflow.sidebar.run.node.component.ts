import {Component, Input, ViewChild} from '@angular/core';
import {Router} from '@angular/router';
import {Project} from '../../../../../model/project.model';
import {
    Workflow,
    WorkflowNode
} from '../../../../../model/workflow.model';
import {WorkflowRun, WorkflowNodeRun} from '../../../../../model/workflow.run.model';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {WorkflowNodeRunParamComponent} from '../../../../../shared/workflow/node/run/node.run.param.component';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {DurationService} from '../../../../../shared/duration/duration.service';
import {Subscription} from 'rxjs/Subscription';
import {first} from 'rxjs/operators';
import 'rxjs/add/observable/zip';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';

@Component({
    selector: 'app-workflow-sidebar-run-node',
    templateUrl: './workflow.sidebar.run.node.component.html',
    styleUrls: ['./workflow.sidebar.run.node.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunNodeComponent {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    currentWorkflowRunSub: Subscription;
    currentWorkflowRun: WorkflowRun;

    currentNodeRunSub: Subscription;
    currentWorkflowNodeRun: WorkflowNodeRun;

    node: WorkflowNode;

    // Modal
    @ViewChild('workflowRunNode')
    workflowRunNode: WorkflowNodeRunParamComponent;
    loading = true;

    displayEditOption = false;
    displaySummary = true;
    duration: string;
    canBeRun = false;
    pipelineStatusEnum = PipelineStatus;

    constructor(private _wrService: WorkflowRunService, private _router: Router,
               private _durationService: DurationService,
                private _workflowEventStore: WorkflowEventStore) {
        this.currentWorkflowRunSub = this._workflowEventStore.selectedRun().subscribe(wr => {
            if (!wr) {
                this.currentWorkflowNodeRun = null;
                return;
            }
            this.currentWorkflowRun = wr;
            this.loading = false;

            // If not the same run => display loading
            if (this.currentWorkflowRun && this.currentWorkflowRun && this.currentWorkflowNodeRun.num !== this.currentWorkflowRun.num) {
                this.loading = true;
            } else {
                this.refreshData();
            }
        });

        this.currentNodeRunSub = this._workflowEventStore.selectedNodeRun()
            .subscribe((nodeRun) => {
                if (!nodeRun) {
                    return;
                }
                this.loading = false;
                this.duration = this.getDuration();
                this.canBeRun = this.getCanBeRun();
                this.currentWorkflowNodeRun = nodeRun;

                // If not the same run => display loading
                if (this.currentWorkflowRun && this.currentWorkflowRun && this.currentWorkflowNodeRun.num !== this.currentWorkflowRun.num) {
                    this.loading = true;
                } else {
                    this.refreshData();
                }
            });
    }

    refreshData(): void {
        this.node = Workflow.getNodeByID(this.currentWorkflowNodeRun.workflow_node_id, this.currentWorkflowRun.workflow);
        this.displayEditOption = this.node != null;
        this.canBeRun = this.getCanBeRun();
    }

    displayLogs() {
        let pip = this.node.pipeline.name;
        this._router.navigate([
            '/project', this.project.key,
            'workflow', this.currentWorkflowRun.workflow.name,
            'run', this.currentWorkflowRun.num,
            'node', this.currentWorkflowNodeRun.id], {queryParams: {name: pip}});
    }

    getDuration() {
        if (!this.currentWorkflowNodeRun) {
            return;
        }
        let done = new Date(this.currentWorkflowNodeRun.done);
        if (PipelineStatus.isActive(this.currentWorkflowNodeRun.status)) {
            done = new Date();
        }

        return this._durationService.duration(new Date(this.currentWorkflowNodeRun.start), done);
    }

    getCanBeRun(): boolean {
        /**
        let appForbid = this.node && this.node.context.application && this.node.context.application.permission &&
            this.node.context.application.permission < PermissionValue.READ_EXECUTE;
        let envForbid = this.node && this.node.context.environment && this.node.context.environment.permission
            && this.node.context.environment.permission < PermissionValue.READ_EXECUTE;

        if (this.workflow.permission < PermissionValue.READ_EXECUTE || appForbid || envForbid) {
          return false;
        }
        if (this.currentWorkflowNodeRun && this.currentWorkflowRun) {
            let nodesRun = this.currentWorkflowRun.nodes[this.currentWorkflowNodeRun.workflow_node_id];
            let nodeRun = nodesRun.find( n => {
               return n.id === this.currentWorkflowNodeRun.id;
            });
            return nodeRun.can_be_run;
        }

        let workflowRunIsNotActive = this.currentWorkflowRun && !PipelineStatus.isActive(this.currentWorkflowRun.status);
        if (workflowRunIsNotActive && this.currentWorkflowNodeRun) {
            return true;
        }

        if (workflowRunIsNotActive && !this.currentWorkflowNodeRun && this.nodeId === this.workflow.root_id) {
            return true;
        }

        let nbNodeFound = 0;
        let parentNodes = Workflow.getParentNodeIds(this.workflow, this.nodeId);
        for (let parentNodeId of parentNodes) {
            for (let nodeRunId in this.currentWorkflowRun.nodes) {
                if (!this.currentWorkflowRun.nodes[nodeRunId]) {
                    continue;
                }
                let nodeRuns = this.currentWorkflowRun.nodes[nodeRunId];
                if (nodeRuns[0].workflow_node_id === parentNodeId) { // if node id is still the same
                    if (PipelineStatus.isActive(nodeRuns[0].status)) {
                        return false;
                    }
                    nbNodeFound++;
                } else if (!Workflow.getNodeByID(nodeRuns[0].workflow_node_id, this.workflow)) {
                    // workflow updated so prefer return true
                    return true;
                }
            }
        }

        if (nbNodeFound !== parentNodes.length) { // It means that a parent node isn't already executed
            return false;
        }
        */
        return true;
    }

    stopNodeRun(): void {
        this.loading = true;
        this._wrService.stopNodeRun(this.project.key, this.currentWorkflowRun.workflow.name,
            this.currentWorkflowRun.num, this.currentWorkflowNodeRun.id)
            .pipe(first())
            .subscribe(() => {
                this._router.navigate([
                    '/project', this.project.key,
                    'workflow', this.currentWorkflowRun.workflow.name,
                    'run', this.currentWorkflowRun.num]);
            });
    }

    openRunNode(): void {
        this.workflowRunNode.show();
    }
}
