import {Component, Input} from '@angular/core';
import {PipelineBuild} from '../../../model/pipeline.model';

@Component({
    selector: 'app-run-summary',
    templateUrl: './run.summary.html',
    styleUrls: ['./run.summary.scss']
})
export class RunSummaryComponent {

    @Input() currentBuild: PipelineBuild;
    @Input() duration: string;

    constructor() { }
}
