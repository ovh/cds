import {Component, Input} from '@angular/core';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-project-workflows',
    templateUrl: './project.workflows.html',
    styleUrls: ['./project.workflows.scss']
})
export class ProjectWorkflowListComponent {

    @Input() project: Project;

    constructor() { }
}
