import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit } from '@angular/core';
import { TestCase, Tests } from 'app/model/pipeline.model';
import { NzFormatEmitEvent, NzTreeNodeOptions } from 'ng-zorro-antd/tree';
import { WorkflowRunResultDetail } from '../../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model';

@Component({
    selector: 'app-run-result-tests',
    templateUrl: './run-result-tests.html',
    styleUrls: ['./run-result-tests.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RunResultTestsComponent implements OnInit, OnChanges {
    @Input() detail: WorkflowRunResultDetail;

    tests: Tests;
    searchValue = '';
    filterModified: boolean;
    activeFilters: Array<string>;
    filterOptions = [];
    nodes = [];

    updateFilters(event): void {
        this.filterOptions = event;
        this.filterModified = !this.filterOptions.map(o => o.checked).reduce((p, c) => p && c);
        this.activeFilters = this.filterOptions.filter(o => o.checked).map(o => o.value);
        this.initTestTree();
        this._cd.markForCheck();
    }

    constructor(
        private _cd: ChangeDetectorRef
    ) {
        this.filterOptions = [
            { label: 'Success', value: 'success', checked: true },
            { label: 'Failed', value: 'failed', checked: true },
            { label: 'Skipped', value: 'skipped', checked: true }
        ];
        this.activeFilters = this.filterOptions.filter(o => o.checked).map(o => o.value);
    }

    ngOnInit(): void { }

    ngOnChanges(): void {
        this.tests = <Tests>{
            ko: 0,
            ok: this.detail.data.tests_stats.ok ?? 0,
            skipped: 0,
            total: this.detail.data.tests_stats.total ?? 0,
            test_suites: this.detail.data.tests_suites.test_suites
        };

        this.initTestTree();

        this._cd.markForCheck();
    }

    initTestTree(): void {
        const nodes = this.tests.test_suites.map(ts => {
            let node = <NzTreeNodeOptions>{
                title: ts.name,
                key: ts.name,
                children: [],
                expanded: true
            };

            const filteredTests = ts.tests.filter(t => this.activeFilters.indexOf(this.computeTestCaseStatus(t)) !== -1);

            // Try to aggregate tests sub runs
            node.children = filteredTests.filter(t => t.name.indexOf('/') === -1).map(t => (<NzTreeNodeOptions>{
                title: t.name,
                key: t.name,
                children: []
            }));
            filteredTests.filter(t => t.name.indexOf('/') !== -1).filter(t => {
                const split = t.name.split(('/'));
                const parentIdx = node.children.findIndex(c => c.key === split[0]);
                if (parentIdx >= 0) {
                    node.children[parentIdx].children.push(<NzTreeNodeOptions>{
                        title: split[1],
                        key: split[1],
                        isLeaf: true
                    })
                } else {
                    node.children.push(<NzTreeNodeOptions>{
                        title: t.name,
                        key: t.name
                    })
                }
            });

            node.children.map(c => {
                c.isLeaf = c.children.length === 0;
                return c;
            }).sort((a, b) => a.key < b.key ? -1 : 1);

            return node;
        });

        this.nodes = nodes.filter(n => (n.children.length > 0 && this.filterModified) || !this.filterModified);
    }

    nzEvent(event: NzFormatEmitEvent): void { }

    computeTestCaseStatus(tc: TestCase): string {
        if (!tc.errors && !tc.failures && !tc.skipped) {
            return 'success';
        } else if (tc.errors?.length > 0 || tc.failures?.length > 0) {
            return 'failed';
        } else {
            return 'skipped';
        }
    }

}
