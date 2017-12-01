import {Component, Input} from '@angular/core';
import {Tests} from '../../../model/pipeline.model';

@Component({
    selector: 'app-tests-result',
    templateUrl: './tests.result.html',
    styleUrls: ['./tests.result.scss']
})
export class TestsResultComponent {

    @Input() tests: Tests;

    filter = 'error';

    constructor() { }
}
