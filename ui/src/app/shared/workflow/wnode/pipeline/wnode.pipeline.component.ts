import { Component, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';
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
        private _router: Router
    ) { }

    ngOnInit(): void {
        this.subSelectedNode = this._workflowEventStore.selectedNode().subscribe(n => {
            this.selected = n && (n.id === this.node.id);
        });
    }

    displayLogs() {
        this._router.navigate([
            '/project', this.project.key,
            'workflow', this.workflow.name,
            'run', this.noderun.num,
            'node', this.noderun.id], { queryParams: { name: this.node.name } });
    }
}
