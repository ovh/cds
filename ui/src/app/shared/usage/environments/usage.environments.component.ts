import {Component, Input} from '@angular/core';
import {Environment} from '../../../model/environment.model';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-usage-environments',
    templateUrl: './usage.environments.html'
})
export class UsageEnvironmentsComponent {

    @Input() project: Project;
    @Input() environments: Array<Environment>;

    constructor() { }
}
