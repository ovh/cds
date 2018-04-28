import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {Router, ActivatedRoute} from '@angular/router';
import {RouterService} from '../../../../../service/router/router.service';
import {Project} from '../../../../../model/project.model';
import {PermissionValue} from '../../../../../model/permission.model';
import {
    Workflow,
    WorkflowNode
} from '../../../../../model/workflow.model';
import {WorkflowRun, WorkflowNodeRun} from '../../../../../model/workflow.run.model';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {WorkflowNodeRunParamComponent} from '../../../../../shared/workflow/node/run/node.run.param.component';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {WorkflowCoreService} from '../../../../../service/workflow/workflow.core.service';
import {DurationService} from '../../../../../shared/duration/duration.service';
import {Subscription} from 'rxjs/Subscription';
import {first} from 'rxjs/operators';
import 'rxjs/add/observable/zip';

@Component({
    selector: 'app-workflow-sidebar-run-node',
    templateUrl: './workflow.sidebar.run.node.component.html',
    styleUrls: ['./workflow.sidebar.run.node.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunNodeComponent implements OnInit {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() number: number;
    @Input() open: boolean;

    // Modal
    @ViewChild('workflowRunNode')
    workflowRunNode: WorkflowNodeRunParamComponent;
    node: WorkflowNode;
    nodeId: number;
    runId: number;
    runNumber: number;
    currentWorkflowRunSub: Subscription;
    currentNodeRunSub: Subscription;
    queryParamsSub: Subscription;
    loading = true;
    currentWorkflowRun: WorkflowRun;
    currentWorkflowNodeRun: WorkflowNodeRun;
    displayEditOption = false;
    displaySummary = true;
    duration: string;
    canBeRun = false;
    pipelineStatusEnum = PipelineStatus;

    constructor(private _wrService: WorkflowRunService, private _router: Router,
      private _workflowCoreService: WorkflowCoreService, private _activatedRoute: ActivatedRoute,
      private _durationService: DurationService, private _routerService: RouterService) {

    }

    ngOnInit() {
        this.queryParamsSub = this._activatedRoute.queryParams.subscribe((queryparams) => {
          this.runId = Number.isNaN(queryparams['selectedNodeRunId']) ? null : parseInt(queryparams['selectedNodeRunId'], 10);
          this.runNumber = Number.isNaN(queryparams['selectedNodeRunNum']) ? null : parseInt(queryparams['selectedNodeRunNum'], 10);
          this.nodeId = Number.isNaN(queryparams['selectedNodeId']) ? null : parseInt(queryparams['selectedNodeId'], 10);
          this.displaySummary = this.runId !== -1;

          if (!this.currentWorkflowRun) {
            this.currentWorkflowNodeRun = null;
            return;
          }

          this.node = Workflow.getNodeByID(this.nodeId, this.currentWorkflowRun.workflow);
          let wr = this.currentWorkflowRun;
          if (this.node && wr.nodes && wr.nodes[this.node.id] && Array.isArray(wr.nodes[this.node.id])) {
              this.currentWorkflowNodeRun = wr.nodes[this.node.id].find((n) => n.id === this.runId && n.num === this.runNumber);
              this.duration = this.getDuration();
          } else {
              this.currentWorkflowNodeRun = null;
          }
          this.canBeRun = this.getCanBeRun();

          this.displayEditOption = Workflow.getNodeByID(this.nodeId, this.workflow) != null;
        });

        this.currentWorkflowRunSub = this._workflowCoreService.getCurrentWorkflowRun().subscribe((wr) => {
            if (!wr) {
                this.currentWorkflowNodeRun = null;
                return;
            }
            this.currentWorkflowRun = wr;
            this.node = Workflow.getNodeByID(this.nodeId, this.currentWorkflowRun.workflow);

            if (this.node && wr.nodes && wr.nodes[this.node.id] && Array.isArray(wr.nodes[this.node.id])) {
                this.currentWorkflowNodeRun = wr.nodes[this.node.id].find((n) => n.id === this.runId && n.num === this.runNumber);
                this.duration = this.getDuration();
            } else {
                this.currentWorkflowNodeRun = null;
            }

            this.loading = false;

            this.displayEditOption = Workflow.getNodeByID(this.nodeId, this.workflow) != null;
            this.canBeRun = this.getCanBeRun();
          }
        );

        this.currentNodeRunSub = this._workflowCoreService.getCurrentNodeRun()
          .subscribe((nodeRun) => {
            let params = this._routerService.getRouteParams({}, this._activatedRoute);
            if (!nodeRun || !params['nodeId']) {
              return;
            }
            this.loading = false;
            this.currentWorkflowNodeRun = nodeRun;
            this.duration = this.getDuration();
            this.canBeRun = this.getCanBeRun();

            // If not the same run => display loading
            if (this.currentWorkflowNodeRun.num !== this.number) {
                this.loading = true;
            }
          });
    }

    displayLogs() {
        let pip = this.node.pipeline.name;
        this._router.navigate([
            '/project', this.project.key,
            'workflow', this.workflow.name,
            'run', this.runNumber,
            'node', this.runId], {queryParams: {name: pip}});
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

        return true;
    }

    stopNodeRun(): void {
        this.loading = true;
        this._wrService.stopNodeRun(this.project.key, this.workflow.name, this.runNumber, this.runId)
            .pipe(first())
            .subscribe(() => {
                this.currentWorkflowNodeRun.status = PipelineStatus.STOPPED;
                this.currentWorkflowRun.status = PipelineStatus.STOPPED;
                this._workflowCoreService.setCurrentWorkflowRun(this.currentWorkflowRun);
                this._router.navigate([
                    '/project', this.project.key,
                    'workflow', this.workflow.name,
                    'run', this.runNumber]);
            });
    }

    openRunNode(): void {
        this.workflowRunNode.show();
    }
}
