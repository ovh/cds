import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode-fork',
    templateUrl: './node.fork.html',
    styleUrls: ['./node.fork.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeForkComponent {
    @Input() public project: Project;
    @Input() public node: WNode;
    @Input() public workflow: Workflow;
    @Input() public noderun: WorkflowNodeRun;
    @Input() public selected: boolean;

    pipelineStatus = PipelineStatus;
}
