import {AfterViewInit, Component, ElementRef, Input, OnInit} from '@angular/core';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {WNode, Workflow} from '../../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../../model/workflow.run.model';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode-join',
    templateUrl: './node.join.html',
    styleUrls: ['./node.join.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeJoinComponent implements OnInit, AfterViewInit {

    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public workflowrun: WorkflowRun;
    @Input() public selected: boolean;

    canRun: boolean;
    pipelineStatusEnum = PipelineStatus;

    elementRef: ElementRef;

    constructor(elt: ElementRef) {
        this.elementRef = elt;
    }

    ngOnInit(): void {
        this.canBeLaunched();
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
    }

    canBeLaunched() {
        if (!this.workflowrun || !this.workflowrun.nodes) {
            return false;
        }
        let lengthParentRun = 0;
        Object.keys(this.workflowrun.nodes).forEach((key) => {
            if (this.workflowrun.nodes[key].length &&
                this.node.parents.findIndex(p => p.parent_id === this.workflowrun.nodes[key][0].workflow_node_id) !== -1) {
                lengthParentRun++;
            }
        });
        this.canRun = this.node.parents.length === lengthParentRun;
    }
}
