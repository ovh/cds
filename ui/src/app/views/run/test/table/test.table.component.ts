import { Component, Input } from '@angular/core';
import * as AU from 'ansi_up';
import { TestCase, TestSuite } from '../../../../model/pipeline.model';
import { Table } from '../../../../shared/table/table';

@Component({
    selector: 'app-test-table',
    templateUrl: './test.table.html',
    styleUrls: ['./test.table.scss']
})
export class TestTableComponent extends Table<TestCase> {

    filteredTests: Array<TestCase>;
    filter: string;
    ansi_up = new AU.default;

    @Input() tests: Array<TestSuite>;
    @Input('statusFilter')
    set statusFilter(status: string) {
        this.filter = status;
        this.updateFilteredTests();
    }

    constructor() {
        super();
        this.nbElementsByPage = 20;
    }

    getData(): Array<TestCase> {
        if (!this.filteredTests) {
            this.updateFilteredTests();
        }
        return this.filteredTests;
    }

    updateFilteredTests(): void {
        this.filteredTests = new Array<TestCase>();
        if (!this.tests) {
            return;
        }
        switch (this.filter) {
            case 'error':
                for (let ts of this.tests) {
                    if (ts.errors > 0 || ts.failures > 0) {
                        let testCases = ts.tests
                            .filter(tc => (tc.errors && tc.errors.length > 0) || (tc.failures && tc.failures.length > 0))
                            .map(tc => {
                                tc.fullname = ts.name + ' / ' + tc.name;
                                return tc;
                            });
                        this.filteredTests.push(...testCases);
                    }
                };
                break;
            case 'skipped':
                for (let ts of this.tests) {
                    if (ts.skipped > 0) {
                        let testCases = ts.tests
                            .filter(tc => (tc.skipped && tc.skipped.length > 0))
                            .map(tc => {
                                tc.fullname = ts.name + ' / ' + tc.name;
                                return tc;
                            });
                        this.filteredTests.push(...testCases);
                    }
                };
                break;
            default:
                for (let ts of this.tests) {
                    let testCases = ts.tests.map(tc => {
                        tc.fullname = ts.name + ' / ' + tc.name;
                        return tc;
                    });
                    this.filteredTests.push(...testCases);
                }
        }
    }

    getLogs(logs) {
        if (logs && logs.value) {
            return this.ansi_up.ansi_to_html(logs.value);
        }
        return '';
    }
}
