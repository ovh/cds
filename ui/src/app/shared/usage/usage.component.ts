import {Component, Input} from '@angular/core';
import {Workflow} from '../../model/workflow.model';
import {Application} from '../../model/application.model';
import {Pipeline} from '../../model/pipeline.model';
import {Environment} from '../../model/environment.model';
import {Project} from '../../model/project.model';
import {User} from '../../model/user.model';

@Component({
    selector: 'app-usage',
    templateUrl: './usage.component.html',
    styleUrls: ['./usage.component.scss']
})
export class UsageComponent {

    @Input() project: Project;
    @Input() workflows: Array<Workflow>;
    @Input() applications: Array<Application>;
    @Input() pipelines: Array<Pipeline>;
    @Input() environments: Array<Environment>;

    constructor() { }
}
