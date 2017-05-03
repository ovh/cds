import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {WorkflowNode, WorkflowNodeContext} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-node-form',
    templateUrl: './workflow.node.form.html',
    styleUrls: ['./workflow.node.form.scss']
})
export class WorkflowNodeItemFormComponent {

    @Input() project: Project;
    @Input() node: WorkflowNode;
    @Output() nodeChange = new EventEmitter<WorkflowNode>();

    constructor() { }

    updatePipeline(pipelineName: string): void {
        this.node.pipeline_id = this.project.pipelines.find( p => p.name === pipelineName).id;
        this.nodeChange.emit(this.node);
    }

    updateApplication(appName: string): void {
        if (!this.node.context) {
            this.node.context = new WorkflowNodeContext();
        }
        this.node.context.application_id = this.project.applications.find(a => a.name === appName).id;
        this.nodeChange.emit(this.node);
    }

    updateEnvironment(envName: string): void {
        if (!this.node.context) {
            this.node.context = new WorkflowNodeContext();
        }
        this.node.context.environment_id = this.project.environments.find(e => e.name === envName).id;
        this.nodeChange.emit(this.node);
    }
}
