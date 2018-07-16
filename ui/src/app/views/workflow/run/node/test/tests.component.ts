import {Component, Input} from '@angular/core';
import {Coverage} from '../../../../../model/coverage.model';
import {Tests} from '../../../../../model/pipeline.model';

@Component({
    selector: 'app-workflow-tests-result',
    templateUrl: './tests.result.html',
    styleUrls: ['./tests.result.scss']
})
export class WorkflowRunTestsResultComponent {

    @Input() tests: Tests;
    @Input() coverage: Coverage;

    filter = 'error';

    constructor() { }
}
