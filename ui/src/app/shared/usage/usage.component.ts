import {Component, Input} from '@angular/core';
import {Application} from '../../model/application.model';
import {Environment} from '../../model/environment.model';
import {Pipeline} from '../../model/pipeline.model';
import {Project} from '../../model/project.model';
import {Workflow} from '../../model/workflow.model';

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
