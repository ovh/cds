import {Component, Input} from '@angular/core';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-project-workflows',
    templateUrl: './workflow.list.html',
    styleUrls: ['./workflow.list.scss']
})
export class ProjectWorkflowListComponent {

    @Input() project: Project;

    constructor() { }
}
