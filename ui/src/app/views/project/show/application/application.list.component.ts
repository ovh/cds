import { Component, Input } from '@angular/core';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-project-applications',
    templateUrl: './application.list.html',
    styleUrls: ['./application.list.scss']
})
export class ProjectApplicationListComponent {

    @Input() project: Project;

    constructor() { }
}
