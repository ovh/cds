import {Component, Input} from '@angular/core';
import {Workflow} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-usage-workflows',
    templateUrl: './usage.workflows.html'
})
export class UsageWorkflowsComponent {

    @Input() project: Project;
    @Input() workflows: Array<Workflow>;

    constructor() { }
}
