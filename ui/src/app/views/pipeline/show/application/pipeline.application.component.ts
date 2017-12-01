import {Component, Input} from '@angular/core';
import {Application} from '../../../../model/application.model';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-pipeline-application',
    templateUrl: './pipeline.application.html',
    styleUrls: ['./pipeline.application.scss']
})
export class PipelineApplicationComponent {

    @Input() project: Project;
    @Input() applications: Array<Application>;

    constructor() { }
}
