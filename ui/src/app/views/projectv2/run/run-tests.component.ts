import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, ViewChild } from "@angular/core";
import { TestCase, Tests } from "app/model/pipeline.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { NzFormatEmitEvent, NzTreeComponent, NzTreeNodeOptions } from "ng-zorro-antd/tree";

@Component({
	selector: 'app-run-tests',
	templateUrl: './run-tests.html',
	styleUrls: ['./run-tests.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunTestsComponent implements OnChanges {
	@ViewChild('tree') tree: NzTreeComponent;

	@Input() tests: Tests;
	@Output() onSelectTest = new EventEmitter<TestCase>();

	searchValue = '';
	filterModified: boolean;
	filtered: boolean;
	activeFilters: Array<string>;
	filterOptions = [];
	nodes = [];

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
			let node = <NzTreeNodeOptions>{
				title: `${ts.name} - ${ts.time}s`,
				key: ts.name,
				children: [],
				expanded: false,
				selectable: false
			};

			const filteredTests = (ts.tests ?? []).filter(t => this.activeFilters.indexOf(this.computeTestCaseStatus(t)) !== -1);

			// Try to aggregate tests sub runs
			node.children = filteredTests.filter(t => t.name.indexOf('/') === -1).map(t => {
				const status = this.computeTestCaseStatus(t);
				let icon = 'check';
				switch (status) {
					case 'skipped':
						icon = 'warning';
						break;
					case 'failed':
						icon = 'close';
						break;
				}
				return <NzTreeNodeOptions>{
					title: `${t.name} - ${t.time}s`,
					key: `${ts.name}/${t.name}`,
					children: [],
					icon,
					testCase: t
				};
			});
			filteredTests.filter(t => t.name.indexOf('/') !== -1).forEach(t => {
				const split = t.name.split(('/'));
				const parentIdx = node.children.findIndex(c => c.key === split[0]);

				const status = this.computeTestCaseStatus(t);
				let icon = 'check';
				switch (status) {
					case 'skipped':
						icon = 'warning';
						break;
					case 'failed':
						icon = 'close';
						break;
				}

				if (parentIdx >= 0) {
					node.children[parentIdx].children.push(<NzTreeNodeOptions>{
						title: `${split[1]} - ${t.time}s`,
						key: `${ts.name}/${split[0]}/${split[1]}`,
						isLeaf: true,
						icon,
						testCase: t
					})
				} else {
					node.children.push(<NzTreeNodeOptions>{
						title: `${t.name} - ${t.time}s`,
						key: `${ts.name}/${t.name}`,
						isLeaf: true,
						icon,
						testCase: t
					})
				}
			});

			node.children.map(c => {
				c.isLeaf = !c.children || c.children.length === 0;
				return c;
			}).sort((a, b) => a.key < b.key ? -1 : 1);

			return node;
		});

		this.nodes = nodes.filter(n => (n.children.length > 0 && this.filtered) || !this.filtered);
	}

	updateFilters(event): void {
		this.filterModified = true;
		this.filterOptions = event;
		this.filtered = !this.filterOptions.map(o => o.checked).reduce((p, c) => p && c);
		this.activeFilters = this.filterOptions.filter(o => o.checked).map(o => o.value);
		this.initTestTree();
		this._cd.markForCheck();
	}

	nzEvent(event: NzFormatEmitEvent): void {
		if (!event.node) {
			return;
		}
		const recursiveSelectNode = (n: NzTreeNodeOptions): NzTreeNodeOptions => {
			let copy = <NzTreeNodeOptions>{ ...n };
			if (event.node.isLeaf) {
				copy.selected = false;
			}
			if (copy.key === event.node.key) {
				if (copy.children && copy.children.length > 0) {
					copy.expanded = !copy.expanded;
				} else {
					copy.selected = true;
				}
			}
			if (copy.children) {
				copy.children = copy.children.map(n => recursiveSelectNode(n));
			}
			return copy;
		};
		this.nodes = this.nodes.map(n => recursiveSelectNode(n));
		if (event.node.isLeaf) {
			this.onSelectTest.emit(event.node.origin.testCase);
		}
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
}