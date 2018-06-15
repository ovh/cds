import {Component, Input} from '@angular/core';
import {Project} from '../../../model/project.model';
import {Workflow} from '../../../model/workflow.model';

@Component({
    selector: 'app-usage-workflows',
    templateUrl: './usage.workflows.html'
})
export class UsageWorkflowsComponent {

    @Input() project: Project;
    @Input() workflows: Array<Workflow>;

    constructor() { }
}
