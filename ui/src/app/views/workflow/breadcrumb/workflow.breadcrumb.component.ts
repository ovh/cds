import {Component, Input} from '@angular/core';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-worflow-breadcrumb',
    templateUrl: './breadcrumb.html'
})
export class WorkflowBreadCrumbComponent {

    @Input() project: Project;
    @Input() workflowName: string;
    @Input() run: number;
    @Input() nodeName: string;

    constructor() { }
}
