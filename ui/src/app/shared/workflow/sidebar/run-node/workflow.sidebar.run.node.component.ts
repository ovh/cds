import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input,
    OnDestroy,
    OnInit,
    ViewChild
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Select } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, WNodeType, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { WorkflowNodeRunParamComponent } from 'app/shared/workflow/node/run/node.run.param.component';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import 'rxjs/add/observable/zip';
import { first } from 'rxjs/operators';


@Component({
    selector: 'app-workflow-sidebar-run-node',
    templateUrl: './workflow.sidebar.run.node.component.html',
    styleUrls: ['./workflow.sidebar.run.node.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowSidebarRunNodeComponent implements OnDestroy, OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;

    @Select(WorkflowState.getSelectedNode()) node$: Observable<WNode>;
    nodeSubs: Subscription;
    node: WNode;

    @Select(WorkflowState.getSelectedWorkflowRun()) workflowRun$: Observable<WorkflowRun>;
    workflowRunSubs: Subscription;
    currentWorkflowNodeRun: WorkflowNodeRun;

    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;
    workflowRun: WorkflowRun;

    // Loadder for button
    loading: false;
    runNumber: number;
    storeSub: Subscription;
    wNodeType = WNodeType;

    // Modal
    @ViewChild('workflowRunNode', {static: false})
    workflowRunNode: WorkflowNodeRunParamComponent;

    duration: string;
    canBeRun = false;
    pipelineStatusEnum = PipelineStatus;

    durationIntervalID: number;

    constructor(
        private _wrService: WorkflowRunService,
        private _router: Router,
        private _activatedRoute: ActivatedRoute,
        private _durationService: DurationService,
        private _cd: ChangeDetectorRef
    ) {
        this._activatedRoute.params.subscribe(p => {
            this.runNumber = p['number'];
        });
    }

    ngOnInit(): void {
        this.nodeSubs = this.node$.subscribe(n => {
            if (!n && !this.node) {
                return;
            }
            if (n && this.node && n.id === this.node.id) {
                return;
            }
            console.log('REFRESH SIDEBAR NODE');
            this.node = n;
            // Check is the node can be run
            this.refresh();
        });

        this.nodeRunSubs = this.nodeRun$.subscribe( nrs => {
            if (!nrs && !this.currentWorkflowNodeRun) {
                return;
            }
            console.log('REFRESH SIDEBAR NODERUN');
            this.currentWorkflowNodeRun = nrs;

            // Run interval to compute duration
            this.runDurationLoop();
            this.refresh();
        });

        this.workflowRunSubs = this.workflowRun$.subscribe(wrs => {
            if (!wrs && !this.workflowRun) {
                return;
            }
            this.workflowRun = wrs;
            this.refresh();
        });
    }

    refresh(): void {
        this.canBeRun = this.getCanBeRun();
        this._cd.markForCheck();
    }

    displayLogs() {
        if (this.node.type === WNodeType.OUTGOINGHOOK) {
            if (this.currentWorkflowNodeRun && this.node && this.node.outgoing_hook
                && this.node.outgoing_hook.config['target_workflow']) {
                this._router.navigate([
                    '/project', this.project.key,
                    'workflow', this.node.outgoing_hook.config['target_workflow'].value,
                    'run', this.currentWorkflowNodeRun.callback.workflow_run_number
                ], { queryParams: {} });
            }
            return;
        }
        this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name,
                'run', this.runNumber,
                'node', this.currentWorkflowNodeRun.id], { queryParams: { name: this.node.name } });
    }

    runDurationLoop(): void {
        this.deleteInverval();
        this.duration = this.getDuration();

        // Only when pipeline is running
        if (this.currentWorkflowNodeRun && PipelineStatus.isActive(this.currentWorkflowNodeRun.status)) {
            this.durationIntervalID = window.setInterval(() => {
                this.duration = this.getDuration();
            }, 5000);
        }
    }

    getDuration() {
        if (!this.currentWorkflowNodeRun) {
            return;
        }
        let done = new Date(this.currentWorkflowNodeRun.done);
        if (PipelineStatus.isActive(this.currentWorkflowNodeRun.status)) {
            done = new Date();
        } else {
            this.deleteInverval();
        }

        return this._durationService.duration(new Date(this.currentWorkflowNodeRun.start), done);
    }

    getCanBeRun(): boolean {
        if (!this.workflow) {
            return false;
        }
        if (!this.workflow.permissions.executable) {
            return false;
        }

        // If we are in a run, check if current node can be run ( compuite by cds api)
        if (this.currentWorkflowNodeRun && this.workflowRun && this.workflowRun.nodes) {
            let nodesRun = this.workflowRun.nodes[this.currentWorkflowNodeRun.workflow_node_id];
            if (nodesRun) {
                let nodeRun = nodesRun.find(n => {
                    return n.id === this.currentWorkflowNodeRun.id;
                });
                if (nodeRun) {
                    return nodeRun.can_be_run;
                }
            }
            return false;
        }

        let workflowRunIsNotActive = this.workflowRun && !PipelineStatus.isActive(this.workflowRun.status);
        if (workflowRunIsNotActive && this.currentWorkflowNodeRun) {
            return true;
        }

        if (this.node && this.workflowRun) {
            if (workflowRunIsNotActive && !this.currentWorkflowNodeRun &&
                this.node.id === this.workflowRun.workflow.workflow_data.node.id) {
                return true;
            }

            if (this.workflowRun) {
                let nbNodeFound = 0;
                let parentNodes = Workflow.getParentNodeIds(this.workflowRun, this.node.id);
                for (let parentNodeId of parentNodes) {
                    for (let nodeRunId in this.workflowRun.nodes) {
                        if (!this.workflowRun.nodes[nodeRunId]) {
                            continue;
                        }
                        let nodeRuns = this.workflowRun.nodes[nodeRunId];
                        if (nodeRuns[0].workflow_node_id === parentNodeId) { // if node id is still the same
                            if (PipelineStatus.isActive(nodeRuns[0].status)) {
                                return false;
                            }
                            nbNodeFound++;
                        } else if (!Workflow.getNodeByID(nodeRuns[0].workflow_node_id, this.workflowRun.workflow)) {
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

    stopNodeRun(): void {
        this._wrService.stopNodeRun(this.project.key, this.workflow.name,
            this.runNumber, this.currentWorkflowNodeRun.id)
            .pipe(first())
            .subscribe(() => {
                this._router.navigate([
                    '/project', this.project.key,
                    'workflow', this.workflow.name,
                    'run', this.runNumber]);
            });
    }

    openRunNode(): void {
        this.workflowRunNode.show();
    }

    ngOnDestroy(): void {
        this.deleteInverval();
    }

    deleteInverval(): void {
        if (this.durationIntervalID) {
            clearInterval(this.durationIntervalID);
            this.durationIntervalID = 0;
        }
    }
}
