import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Failure, TestCase, Tests } from '../../../../../../model/pipeline.model';

import { ThemeStore } from 'app/service/services.module';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-test-table',
    templateUrl: './test.table.html',
    styleUrls: ['./test.table.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowRunTestTableComponent implements OnInit {
    @ViewChild('code', {static: false}) codemirror: any;

    codeMirrorConfig: any;
    columns: Array<Column<TestCase>>;
    filter: Filter<TestCase>;
    filterInput: string;
    testcases: Array<TestCase>;
    testCaseSelected: TestCase;
    testsData: Tests;
    themeSubscription: Subscription;

    statusFilter(status: string) {
        if (status === 'all') {
            this.filterInput = '';
        } else if (status !== '') {
            this.filterInput = 'status:' + status;
        }
    }

    @Input('tests')
    set tests(tests: Tests) {
        this.testcases = new Array<TestCase>();
        this.testsData = tests;
        if (!tests || !tests.test_suites) {
            return
        }
        for (let ts of tests.test_suites) {
            if (ts.tests) {
                let testCases = ts.tests.map(tc => {
                    tc.fullname = ts.name + ' / ' + tc.name;
                    if (!tc.errors && !tc.failures) {
                        tc.status = 'success';
                    } else if ( (tc.errors && tc.errors.length > 0) || (tc.failures && tc.failures.length > 0)) {
                        tc.status = 'failed';
                    } else {
                        tc.status = 'skipped';
                    }

                    return tc;
                });
                this.testcases.push(...testCases);
            }
        }
    }

    ngOnInit(): void {
        this.themeSubscription = this._theme.get()
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    constructor(private _theme: ThemeStore, private _cd: ChangeDetectorRef) {
        this.codeMirrorConfig = {
            lineWrapping: false,
            autoRefresh: true,
            readOnly: true
        };

        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                if (lowerFilter.indexOf('status:') === 0) {
                    return lowerFilter.indexOf(d.status) >= 7;
                }
                return d.fullname.toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };

        this.columns = [
            <Column<TestCase>>{
                type: ColumnType.ICON,
                class: 'one',
                selector: (tc: TestCase) => {
                    if (tc.status === 'success') {
                        return ['green', 'check', 'icon'];
                    } else if (tc.status === 'failed') {
                        return ['red', 'remove', 'circle', 'icon'];
                    }
                    return ['grey', 'ban', 'icon'];
                }
            },
            <Column<TestCase>>{
                class: 'ten',
                selector: (tc: TestCase) => tc.fullname
            }
        ];
    }

    clickTestCase(tc: TestCase): void {
        if (this.testCaseSelected && this.testCaseSelected.fullname === tc.fullname) {
            this.testCaseSelected = undefined;
            return
        }
        tc.errorsAndFailures = this.getFailureString(tc.errors)
        tc.errorsAndFailures += this.getFailureString(tc.failures)
        this.testCaseSelected = tc;
    }

    getFailureString(fs: Array<Failure>): string {
        if (!fs) {
            return '';
        }
        let r = '';
        for (const f of fs) {
            if (f.message) {
                r += f.message + '<hr>';
            }
            if (f.value) {
                r += f.value;
            }
            r += '\n';
        }
        return r;
    }

}
