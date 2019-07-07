import { ChangeDetectionStrategy, Component, Input, OnInit } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowHookModel } from 'app/model/workflow.hook.model';
import { WNode, Workflow, WorkflowNodeHookConfigValue } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode-outgoing-hook',
    templateUrl: './node.outgoing.html',
    styleUrls: ['./node.outgoing.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeOutGoingHookComponent implements OnInit {
    @Input() node: WNode;
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() noderun: WorkflowNodeRun;
    @Input() workflowrun: WorkflowRun;
    @Input() selected: boolean;

    icon: string;
    model: WorkflowHookModel;
    pipelineStatus = PipelineStatus;

    constructor() { }

    ngOnInit(): void {
        this.model = this.workflow.outgoing_hook_models[this.node.outgoing_hook.hook_model_id];
        if (this.node && this.node.outgoing_hook.config['hookIcon'] && this.node.outgoing_hook.config['hookIcon'].value) {
            this.icon = (<WorkflowNodeHookConfigValue>this.node.outgoing_hook.config['hookIcon']).value.toLowerCase();
        } else {
            this.icon = this.model.icon.toLowerCase();
        }
    }
}
