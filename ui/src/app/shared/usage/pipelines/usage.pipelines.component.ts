import {Component, Input} from '@angular/core';
import {Pipeline} from '../../../model/pipeline.model';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-usage-pipelines',
    templateUrl: './usage.pipelines.html'
})
export class UsagePipelinesComponent {

    @Input() project: Project;
    @Input() pipelines: Array<Pipeline>;

    constructor() { }
}
