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
        this.nbElementsByPage = 20;
    }

    getData(): any[] {
        if (!this.filteredTests) {
            this.updateFilteredTests();
        }
        return this.filteredTests;
    }

    updateFilteredTests(): void {
        this.filteredTests = new Array<TestCase>();
        switch (this.filter) {
            case 'error':
                if (this.tests) {
                    this.tests.forEach(ts => {
                        if (ts.errors > 0 || ts.failures > 0) {
                            this.filteredTests.push(...ts.tests.map(tc => {tc.name = ts.name + ' / ' + tc.name; return tc}).filter(tc => {
                                return (tc.errors && tc.errors.length > 0) || (tc.failures && tc.failures.length > 0);
                            }));
                        }
                    });
                }
                break;
            case 'skipped':
            if (this.tests) {
                this.tests.forEach(ts => {
                    if (ts.skipped > 0) {
                        this.filteredTests.push(...ts.tests.map(tc => {tc.name = ts.name + ' / ' + tc.name; return tc}).filter(tc => {
                            return (tc.skipped && tc.skipped.length > 0);
                        }));
                    }
                });
            }
                break;
            default:
                if (this.tests) {
                    this.tests.forEach(ts => {
                        this.filteredTests.push(...ts.tests.map(tc => {tc.name = ts.name + ' / ' + tc.name; return tc}));
                    });
                }
        }
    }

    getLogs(logs) {
        if (logs && logs.value) {
            return ansi_up.ansi_to_html(logs.value);
        }
        return '';
    }
}
