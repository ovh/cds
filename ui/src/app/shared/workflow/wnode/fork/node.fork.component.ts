import {Component, Input} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {WNode, Workflow} from '../../../../model/workflow.model';
import {WorkflowNodeRun} from '../../../../model/workflow.run.model';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode-fork',
    templateUrl: './node.fork.html',
    styleUrls: ['./node.fork.scss']
})
@AutoUnsubscribe()
export class WorkflowWNodeForkComponent {

    @Input() public project: Project;
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public selected: boolean;
}
