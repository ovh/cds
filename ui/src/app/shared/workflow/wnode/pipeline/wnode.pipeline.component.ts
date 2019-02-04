import { Component, Input, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { PipelineStatus } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { WNode, Workflow } from '../../../../model/workflow.model';
import { WorkflowNodeRun } from '../../../../model/workflow.run.model';
import { WorkflowEventStore } from '../../../../service/workflow/workflow.event.store';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode-pipeline',
    templateUrl: './node.pipeline.html',
    styleUrls: ['./node.pipeline.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodePipelineComponent implements OnInit {
    @Input() public project: Project;
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public warnings: number;
    selected: boolean;
    pipelineStatus = PipelineStatus;
    subSelectedNode: Subscription;

    constructor(
        private _workflowEventStore: WorkflowEventStore,
        private _activatedRoute: ActivatedRoute,
        private _router: Router
    ) { }

    ngOnInit(): void {
        this.subSelectedNode = this._workflowEventStore.selectedNode().subscribe(n => {
            this.selected = n && (n.id === this.node.id);
        });
    }

    displayLogs() {
        if (this._workflowEventStore.isRunSelected() && this.noderun) {
            this._router.navigate(['node', this.noderun.id], {
                relativeTo: this._activatedRoute,
                queryParams: {
                    name: this.node.name,
                    node_id: this.node.id, node_ref: this.node.ref
                }
            });
        } else {
            this._router.navigate([
                '/project', this.project.key,
                'pipeline', Workflow.getPipeline(this.workflow, this.node).name
            ], {
                    queryParams: {
                        workflow: this.workflow.name,
                        node_id: this.node.id,
                        node_ref: this.node.ref
                    }
                }
            );
        }
    }
}
