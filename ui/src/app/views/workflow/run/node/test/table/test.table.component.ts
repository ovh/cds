import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Failure, TestCase, Tests } from 'app/model/pipeline.model';
import { ThemeStore } from 'app/service/services.module';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { cloneDeep } from 'lodash-es';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';


@Component({
    selector: 'app-workflow-test-table',
    templateUrl: './test.table.html',
    styleUrls: ['./test.table.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowRunTestTableComponent implements OnInit {
    @ViewChild('codemirror1') codemirror1: any;
    @ViewChild('codemirror2') codemirror2: any;
    @ViewChild('codemirror3') codemirror3: any;

    codeMirrorConfig: any;
    columns: Array<Column<TestCase>>;
    filter: Filter<TestCase>;

    beforeClickFilter: string;
    filterInput: string;
    filterIndex: number;
    countFilteredElement: number;

    testCaseSelected: TestCase;
    themeSubscription: Subscription;

    _tests: Tests;
    testcases: Array<TestCase>;
    @Input()
    set tests(data: Tests) {
        this._tests = cloneDeep(data);
        if (this._tests && this._tests.ko > 0 && (!this.filterInput || this.filterInput === '')) {
            this.statusFilter('failed');
        }
        this.initTestCases(this._tests);
    }
    get tests() {
        return this._tests;
    }

    constructor(private _theme: ThemeStore, private _cd: ChangeDetectorRef) {
        this.codeMirrorConfig = {
            lineWrapping: false,
            lineNumbers: true,
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
            },
            <Column<TestCase>>{
                type: ColumnType.BUTTON,
                name: '',
                class: 'two right aligned',
                selector: (tc: TestCase, index?: number) => ({
                        icon: 'eye',
                        class: 'icon small',
                        click: () => this.clickTestCase(tc, index)
                    }),
            },
        ];
    }

    resetFilter(count: number): void {
        if (this.countFilteredElement !== count) {
            this.countFilteredElement = count;
            delete this.testCaseSelected;
            delete this.beforeClickFilter;
        }
    }

    ngOnInit(): void {
        this.themeSubscription = this._theme.get()
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(t => {
                this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
                if (this.codemirror1 && this.codemirror1.instance) {
                    this.codemirror1.instance.setOption('theme', this.codeMirrorConfig.theme);
                }
                if (this.codemirror2 && this.codemirror2.instance) {
                    this.codemirror2.instance.setOption('theme', this.codeMirrorConfig.theme);
                }
                if (this.codemirror3 && this.codemirror3.instance) {
                    this.codemirror3.instance.setOption('theme', this.codeMirrorConfig.theme);
                }
                this._cd.markForCheck();
            });
    }


    statusFilter(status: string) {
        let newFilter = '';
        if (status !== 'all') {
            newFilter = 'status:' + status;
        }

        if (this.filterInput === newFilter) {
            return;
        }
        this.testCaseSelected = null;
        this.filterInput = newFilter;
        this._cd.markForCheck();
    }

    initTestCases(data: Tests) {
        this.testcases = new Array<TestCase>();
        if (!data || !data.test_suites) {
            return;
        }
        for (let ts of data.test_suites) {
            if (ts.tests) {
                let testCase = ts.tests.map(tc => {
                    tc.fullname = ts.name + ' / ' + tc.name;
                    if (!tc.errors && !tc.failures) {
                        tc.status = 'success';
                    } else if ((tc.errors && tc.errors.length > 0) || (tc.failures && tc.failures.length > 0)) {
                        tc.status = 'failed';
                    } else {
                        tc.status = 'skipped';
                    }

                    return tc;
                });
                this.testcases.push(...testCase);
            }
        }
        this._cd.detectChanges();
    }

    clickTestCase(tc: TestCase, index: number): void {
        if (this.testCaseSelected && this.testCaseSelected.fullname === tc.fullname  && this.filterIndex === index) {
            this.testCaseSelected = undefined;
            this.filterInput = this.beforeClickFilter;
            delete this.beforeClickFilter;
            return
        }
        this.filterIndex = index;
        this.beforeClickFilter = this.filterInput;
        this.filterInput = tc.fullname;
        tc.errorsAndFailures = this.getFailureString(tc.errors);
        tc.errorsAndFailures += this.getFailureString(tc.failures);
        this.testCaseSelected = tc;
        this._cd.markForCheck();
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
