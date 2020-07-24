import { ChangeDetectionStrategy, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowHookModel } from 'app/model/workflow.hook.model';
import { WNode, Workflow, WorkflowNodeHookConfigValue } from 'app/model/workflow.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';

@Component({
    selector: 'app-workflow-wnode-outgoing-hook',
    templateUrl: './node.outgoing.html',
    styleUrls: ['./node.outgoing.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeOutGoingHookComponent implements OnInit, OnDestroy {
    @Input() node: WNode;
    @Input() workflow: Workflow;
    @Input() noderun: WorkflowNodeRun;

    project: Project;

    icon: string;
    model: WorkflowHookModel;
    pipelineStatus = PipelineStatus;

    constructor(private _store: Store) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.model = this.workflow.outgoing_hook_models[this.node.outgoing_hook.hook_model_id] || this.node.outgoing_hook.model;
        if (this.node && this.node.outgoing_hook.config['hookIcon'] && this.node.outgoing_hook.config['hookIcon'].value) {
            this.icon = (<WorkflowNodeHookConfigValue>this.node.outgoing_hook.config['hookIcon']).value.toLowerCase();
        } else {
            this.icon = this.model.icon.toLowerCase();
        }
    }
}
