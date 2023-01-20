import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { Failure, Skipped, TestCase, Tests } from 'app/model/pipeline.model';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { PreferencesState } from 'app/store/preferences.state';
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

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
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
                if (lowerFilter.indexOf('fullname:') === 0) {
                    return d.fullname.toLowerCase() === lowerFilter.substring(9);
                }
                return d.fullname.toLowerCase().indexOf(lowerFilter) !== -1;
            };
        };

        this.columns = [
            <Column<TestCase>>{
                type: ColumnType.ICON,
                selector: (tc: TestCase) => ({
                    iconTheme: 'outline',
                    iconType: (() => {
                        if (tc.status === 'success') {
                            return 'check';
                        } else if (tc.status === 'failed') {
                            return 'stop';
                        } else {
                            return 'stop';
                        }
                    })(),
                    iconColor: (() => {
                        if (tc.status === 'success') {
                            return 'green';
                        } else if (tc.status === 'failed') {
                            return 'red';
                        } else {
                            return 'grey';
                        }
                    })()
                })
            },
            <Column<TestCase>>{
                class: 'ten',
                selector: (tc: TestCase) => tc.fullname
            },
            <Column<TestCase>>{
                type: ColumnType.BUTTON,
                name: 'Action',
                class: 'rightAlign',
                selector: (tc: TestCase) => ({
                    buttonDanger: false,
                    iconType: 'eye',
                    iconTheme: 'outline',
                    click: () => this.clickTestCase(tc)
                })
            },
        ];
    }

    dataChanged(count: number): void {
        if (count !== 1) {
            delete this.testCaseSelected;
            this._cd.markForCheck();
        }
    }

    filterChanged(value: string): void {
        this.filterInput = value;
    }

    ngOnInit(): void {
        this.themeSubscription = this._store.select(PreferencesState.theme)
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
                    if (!tc.errors && !tc.failures && !tc.skipped) {
                        tc.status = 'success';
                    } else if (tc.errors?.length > 0 || tc.failures?.length > 0) {
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

    clickTestCase(tc: TestCase): void {
        if (this.testCaseSelected && this.testCaseSelected.fullname === tc.fullname) {
            this.filterInput = this.beforeClickFilter;
            delete this.beforeClickFilter;
            delete this.testCaseSelected;
            this._cd.markForCheck();
            return;
        }
        this.beforeClickFilter = this.filterInput;
        this.filterInput = 'fullname:' + tc.fullname;
        tc.messages = this.getFailureString(tc.errors);
        tc.messages += this.getFailureString(tc.failures);
        tc.messages += this.getFailureString(tc.skipped);
        this.testCaseSelected = tc;
        this._cd.markForCheck();
    }

    getFailureString(fs: Array<Failure | Skipped>): string {
        if (!fs) {
            return '';
        }
        return fs.map(f => {
            let r = '';
            if (f.message) {
                r += f.message;
            }
            if (f.value) {
                r += f.value;
            }
            return r;
        }).filter(r => !!r).join('\n');
    }

}
