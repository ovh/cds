import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from "@angular/core";
import { TestCase, Tests } from "app/model/pipeline.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";

@Component({
	selector: 'app-run-tests',
	templateUrl: './run-tests.html',
	styleUrls: ['./run-tests.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunTestsComponent implements OnChanges {
	@Input() tests: Tests;
	@Output() onSelectTest = new EventEmitter<TestCase>();

	searchValue = '';
	filterModified: boolean;
	filtered: boolean;
	activeFilters: Array<string>;
	filterOptions = [];
	nodes = [];
	treeExpandState: { [key: string]: boolean } = {};

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

	ngOnChanges(changes: SimpleChanges): void {
		if (!this.filterModified) {
			if (this.tests.ko > 0 || this.tests.skipped > 0) {
				this.filterOptions = [
					{ label: 'Success', value: 'success', checked: false },
					{ label: 'Failed', value: 'failed', checked: true },
					{ label: 'Skipped', value: 'skipped', checked: true }
				];
				this.filtered = true;
				this.activeFilters = ['failed', 'skipped'];
			}
		}
		this.initTestTree();
		this._cd.markForCheck();
	}

	initTestTree(): void {
		if (!this.tests || !this.tests.test_suites) {
			this.nodes = [];
			return;
		}

		const nodes = this.tests.test_suites.map(ts => {
			let node = {
				title: `${ts.name}`,
				key: ts.name,
				children: [],
				time: `${ts.time}s`
			};

			const filteredTests = (ts.tests ?? []).filter(t => {
				const statusMatch = this.activeFilters.indexOf(this.computeTestCaseStatus(t)) !== -1;
				const nameMatch = !this.searchValue || t.name.toLowerCase().indexOf(this.searchValue.toLowerCase()) !== -1;
				return statusMatch && nameMatch;
			});

			// Try to aggregate tests sub runs
			node.children = filteredTests.filter(t => t.name.indexOf('/') === -1).map(t => {
				return {
					title: `${t.name}`,
					key: `${ts.name}/${t.name}`,
					testCase: t,
					status: this.computeTestCaseStatus(t),
					time: `${t.time}s`
				};
			});
			filteredTests.filter(t => t.name.indexOf('/') !== -1).forEach(t => {
				const split = t.name.split(('/'));
				const parentIdx = node.children.findIndex(c => c.key === split[0]);
				if (parentIdx >= 0) {
					node.children[parentIdx].children.push({
						title: `${split[1]}`,
						key: `${ts.name}/${split[0]}/${split[1]}`,
						testCase: t,
						status: this.computeTestCaseStatus(t),
						time: `${t.time}s`
					})
				} else {
					node.children.push({
						title: `${t.name}`,
						key: `${ts.name}/${t.name}`,
						testCase: t,
						status: this.computeTestCaseStatus(t),
						time: `${t.time}s`
					})
				}
			});

			node.children.map(c => {
				c.isLeaf = !c.children || c.children.length === 0;
				return c;
			}).sort((a, b) => a.key < b.key ? -1 : 1);

			return node;
		});

		this.nodes = nodes.filter(n => (n.children.length > 0 && this.filtered) || !this.filtered).map(n => {
			return {
				...n,
				success: n.children.filter(t => t.status === 'success').length,
				failed: n.children.filter(t => t.status === 'failed').length,
				skipped: n.children.filter(t => t.status === 'skipped').length
			}
		});

		// Sort test cases for each test suites
		for (let i = 0; i < this.nodes.length; i++) {
			this.nodes[i].children.sort((a, b) => {
				return a.title < b.title ? -1 : 1;
			});
		}

		// Sort test suites
		this.nodes.sort((a, b) => {
			if (a.failed !== b.failed) {
				return a.failed > b.failed ? -1 : 1;
			}
			if (a.skipped !== b.skipped) {
				return a.skipped > b.skipped ? -1 : 1;
			}
			if (a.success !== b.success) {
				return a.success > b.success ? -1 : 1;
			}
			return a.title < b.title ? -1 : 1;
		});

		// Expand test suites with failed tests by default
		this.nodes.forEach(s => {
			if (!this.treeExpandState.hasOwnProperty(s.key) && s.failed > 0) {
				this.treeExpandState[s.key] = true;
			}
		});
	}

	updateFilters(event): void {
		this.filterModified = true;
		this.filterOptions = event;
		this.filtered = !this.filterOptions.map(o => o.checked).reduce((p, c) => p && c);
		this.activeFilters = this.filterOptions.filter(o => o.checked).map(o => o.value);
		this.initTestTree();
		this._cd.markForCheck();
	}

	computeTestCaseStatus(tc: TestCase): string {
		if (!tc.errors && !tc.failures && !tc.skipped) {
			return 'success';
		} else if (tc.errors?.length > 0 || tc.failures?.length > 0) {
			return 'failed';
		} else {
			return 'skipped';
		}
	}

	clickTestSuite(key: string): void {
		this.treeExpandState[key] = !this.treeExpandState[key];
		this._cd.markForCheck();
	}

	clickTestCase(t: TestCase): void {
		this.onSelectTest.emit(t);
	}

	updateSearch(value: string): void {
		this.searchValue = value;
		this.initTestTree();
		this._cd.markForCheck();
	}
}
