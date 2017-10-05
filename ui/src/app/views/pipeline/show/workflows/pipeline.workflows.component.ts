import {Component, Input} from '@angular/core';
import {Workflow} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-pipeline-workflows',
    templateUrl: './pipeline.workflows.html',
    styleUrls: ['./pipeline.workflows.scss']
})
export class PipelineWorkflowsComponent {

    @Input() project: Project;
    @Input() workflows: Array<Workflow>;

    constructor() { }
}
