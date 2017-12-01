import {Component, Input} from '@angular/core';
import {Application} from '../../../model/application.model';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-usage-applications',
    templateUrl: './usage.applications.html'
})
export class UsageApplicationsComponent {

    @Input() project: Project;
    @Input() applications: Array<Application>;

    constructor() { }
}
