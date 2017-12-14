import {Component, Input, OnInit, ViewChild, ChangeDetectorRef, EventEmitter} from '@angular/core';
import {Router, ActivatedRoute} from '@angular/router';
import {Project} from '../../../../../model/project.model';
import {
    Workflow,
    WorkflowNode
} from '../../../../../model/workflow.model';
import {WorkflowRun, WorkflowNodeRun} from '../../../../../model/workflow.run.model';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {PipelineStore} from '../../../../../service/pipeline/pipeline.store';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {WorkflowNodeRunParamComponent} from '../../../../../shared/workflow/node/run/node.run.param.component';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {WorkflowService} from '../../../../../service/workflow/workflow.service';
import {WorkflowCoreService} from '../../../../../service/workflow/workflow.core.service';
import {DurationService} from '../../../../../shared/duration/duration.service';
import {Subscription} from 'rxjs/Subscription';
import {Observable} from 'rxjs/Observable';
import {first} from 'rxjs/operators';
import 'rxjs/add/observable/zip';
import {cloneDeep} from 'lodash';

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
    loading = true;
    currentWorkflowRun: WorkflowRun;
    currentWorkflowNodeRun: WorkflowNodeRun;
    displayEditOption = false;
    displaySummary = true;
    pipelineStatusEnum = PipelineStatus;

    constructor(private _wrService: WorkflowRunService, private _wfService: WorkflowService, private _router: Router,
      private _workflowCoreService: WorkflowCoreService, private _activatedRoute: ActivatedRoute,
      private _durationService: DurationService) {

    }

    ngOnInit() {
        this._activatedRoute.queryParams.subscribe((queryparams) => {
          this.runId = Number.isNaN(queryparams['selectedNodeRunId']) ? null : parseInt(queryparams['selectedNodeRunId'], 10);
          this.runNumber = Number.isNaN(queryparams['selectedNodeRunNum']) ? null : parseInt(queryparams['selectedNodeRunNum'], 10);
          this.nodeId = Number.isNaN(queryparams['selectedNodeId']) ? null : parseInt(queryparams['selectedNodeId'], 10);
          this.displaySummary = this.runId !== -1;

          if (!this.currentWorkflowRun) {
            return;
          }

          this.node = Workflow.getNodeByID(this.nodeId, this.currentWorkflowRun.workflow);
          let wr = this.currentWorkflowRun;
          if (this.node && wr.nodes && wr.nodes[this.node.id] && Array.isArray(wr.nodes[this.node.id])) {
              this.currentWorkflowNodeRun = wr.nodes[this.node.id].find((n) => n.id === this.runId && n.num === this.runNumber);
          } else {
              this.currentWorkflowNodeRun = null;
          }

          this.displayEditOption = Workflow.getNodeByID(this.nodeId, this.workflow) != null;
        });

        this.currentWorkflowRunSub = this._workflowCoreService.getCurrentWorkflowRun().subscribe((wr) => {
            if (!wr) {
                return;
            }
            this.currentWorkflowRun = wr;
            this.node = Workflow.getNodeByID(this.nodeId, this.currentWorkflowRun.workflow);
            if (this.node && wr.nodes && wr.nodes[this.node.id] && Array.isArray(wr.nodes[this.node.id])) {
                this.currentWorkflowNodeRun = wr.nodes[this.node.id].find((n) => n.id === this.runId && n.num === this.runNumber);
            } else {
                this.currentWorkflowNodeRun = null;
            }
            this.loading = false;
            this.displayEditOption = Workflow.getNodeByID(this.nodeId, this.workflow) != null;
          }
        );

        this.currentWorkflowRunSub = this._workflowCoreService.getCurrentWorkflowRun()
            .subscribe((wr) => {
                if (!wr) {
                    return;
                }
                this.currentWorkflowRun = wr;
                this.node = Workflow.getNodeByID(this.nodeId, this.currentWorkflowRun.workflow);
                if (this.node && wr.nodes && wr.nodes[this.node.id] && Array.isArray(wr.nodes[this.node.id])) {
                    this.currentWorkflowNodeRun = wr.nodes[this.node.id].find((n) => n.id === this.runId && n.num === this.runNumber);
                } else {
                    this.currentWorkflowNodeRun = null;
                }
                this.loading = false;
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
        return this._durationService.duration(new Date(this.currentWorkflowNodeRun.start), new Date(this.currentWorkflowNodeRun.done));
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
