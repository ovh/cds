import {Component, Input} from '@angular/core';
import {Table} from '../../../../shared/table/table';
import {TestSuite, TestCase} from '../../../../model/pipeline.model';

declare var ansi_up: any;

@Component({
    selector: 'app-test-table',
    templateUrl: './test.table.html',
    styleUrls: ['./test.table.scss']
})
export class TestTableComponent extends Table {

    filteredTests: Array<TestCase>;
    filter: string;

    @Input() tests: Array<TestSuite>;
    @Input('statusFilter')
    set statusFilter(status: string) {
        this.filter = status;
       this.updateFilteredTests();
    }

    constructor() {
        super();
    }

    getData(): any[] {
        if (!this.filteredTests) {
            this.updateFilteredTests();
        }
        return this.filteredTests;
    }

    updateFilteredTests(): void {
        this.filteredTests = new Array<TestCase>();
        if (this.filter === 'error') {
            if (this.tests) {
                this.tests.forEach(ts => {
                    if (ts.errors > 0 || ts.failures > 0) {
                        this.filteredTests.push(...ts.tests.filter(tc => {
                            return (tc.errors && tc.errors.length > 0) || (tc.failures && tc.failures.length > 0);
                        }));
                    }
                });
            }
        }
    }

    getLogs(logs) {
        return ansi_up.ansi_to_html(logs.value);
    }
}
