import {Component, Input} from '@angular/core';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';

@Component({
    selector: 'app-usage-workflows',
    templateUrl: './usage.workflows.html'
})
export class UsageWorkflowsComponent {

    @Input() project: Project;
    @Input() workflows: Array<Workflow>;

    constructor() { }
}
