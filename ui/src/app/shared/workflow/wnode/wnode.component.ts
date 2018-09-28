import {Location} from '@angular/common';
import {Component, Input} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs';
import {PipelineStatus} from '../../../model/pipeline.model';
import {Project} from '../../../model/project.model';
import {WNode, Workflow} from '../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
import {WorkflowEventStore} from '../../../service/workflow/workflow.event.store';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode',
    templateUrl: './workflow.node.html',
    styleUrls: ['./workflow.node.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeComponent {

    @Input() node: WNode;
    @Input() workflow: Workflow;
    @Input() project: Project;

    // Selected node
    selectedNodeID: number;

    // Selected workflow run
    workflowRun: WorkflowRun;
    currentNodeRun: WorkflowNodeRun;
    warnings = 0;

    // Subscription
    subNodeSelected: Subscription;
    subNodeRun: Subscription;
    subWorkflowRun: Subscription;

    constructor(private _activatedRoute: ActivatedRoute, private _router: Router, private _workflowEventStore: WorkflowEventStore,
                private _location: Location) {
        if (_activatedRoute.snapshot.queryParams['node_id']) {
            this.selectedNodeID = parseInt(_activatedRoute.snapshot.queryParams['node_id'], 10);
        }

        // Subscribe to node selection
        this.subNodeSelected = this._workflowEventStore.selectedNode().subscribe(n => {
            if (n) {
                this.selectedNodeID = n.id;
            }
        });

        // Subscribe to node run events
        this.subNodeRun = this._workflowEventStore.nodeRunEvents().subscribe(wnr => {
            if (wnr) {
                if (!this.workflowRun || this.workflowRun.id !== wnr.workflow_run_id) {
                    return;
                }
                if (wnr.workflow_node_id !== this.node.id) {
                    return;
                }
                this.currentNodeRun = wnr;
            }
        });

        // Subscribe to workflow run events
        this.subWorkflowRun = this._workflowEventStore.selectedRun().subscribe(wr => {
            this.warnings = 0;
            if (wr) {
                if (this.workflowRun && this.workflowRun.id !== wr.id) {
                    this.currentNodeRun = null;
                }
                this.workflowRun = wr;
                if (wr.nodes && wr.nodes[this.node.id] && wr.nodes[this.node.id].length > 0) {
                    if (!this.currentNodeRun ||
                        ((new Date(wr.nodes[this.node.id][0].last_modified)) > (new Date(this.currentNodeRun.last_modified)))) {
                        this.currentNodeRun = wr.nodes[this.node.id][0];
                    }
                }
            } else {
                this.workflowRun = null;
            }

            if (this._activatedRoute.snapshot.queryParams['node_id']) {
                this.selectedNodeID = parseInt(this._activatedRoute.snapshot.queryParams['node_id'], 10);
            }

            if (this.node && this.selectedNodeID && this.node.id === this.selectedNodeID) {
                this._workflowEventStore.setSelectedNode(this.node, false);
                this._workflowEventStore.setSelectedNodeRun(this.currentNodeRun, false);
            }


            if (this.currentNodeRun && this.currentNodeRun.status === PipelineStatus.SUCCESS) {
                this.computeWarnings();
            }
        });
    }




    clickOnNode(): void {
        if (this.workflow.previewMode) {
            return;
        }

        if (this.workflowRun) {
            this._workflowEventStore.setSelectedNode(this.node, false);
            this._workflowEventStore.setSelectedRun(this.workflowRun);
            this._workflowEventStore.setSelectedNodeRun(this.currentNodeRun, true);
        } else {
            this._workflowEventStore.setSelectedNode(this.node, true);
        }

        let url = this._router.createUrlTree(['./'], { relativeTo: this._activatedRoute, queryParams: { 'node_id': this.node.id}});
        this._location.go(url.toString());
    }

    dblClickOnNode() {
        if (this._workflowEventStore.isRunSelected() && this.currentNodeRun) {
            this._workflowEventStore.setSelectedNodeRun(this.currentNodeRun);
            this._router.navigate([
                'node', this.currentNodeRun.id
            ], {relativeTo: this._activatedRoute, queryParams: {name: this.node.name}});
        } else {
            this._workflowEventStore.setSelectedNode(this.node, true);
            this._router.navigate([
                '/project', this.project.key,
                'pipeline', Workflow.getPipeline(this.workflow, this.node).name
            ], {queryParams: {workflow: this.workflow.name}});
        }
    }

    computeWarnings() {
        this.warnings = 0;
        if (!this.currentNodeRun || !this.currentNodeRun.stages) {
            return;
        }
        this.currentNodeRun.stages.forEach((stage) => {
            if (Array.isArray(stage.run_jobs)) {
                this.warnings += stage.run_jobs.reduce((fail, job) => {
                    if (!job.job || !Array.isArray(job.job.step_status)) {
                        return fail;
                    }
                    return fail + job.job.step_status.reduce((failStep, step) => {
                        if (step.status === PipelineStatus.FAIL) {
                            return failStep + 1;
                        }
                        return failStep;
                    }, 0);
                }, 0);
            }
        })
    }

}
