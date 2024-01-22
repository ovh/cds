import { HttpClient, HttpHeaders } from "@angular/common/http";
import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, OnInit, ViewChild } from "@angular/core";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom, map } from "rxjs";
import { V2WorkflowRun } from "app/model/v2.workflow.run.model";
import { Project } from "app/model/project.model";
import { Store } from "@ngxs/store";
import { ProjectState } from "app/store/project.state";
import { NzAutocompleteTriggerDirective } from "ng-zorro-antd/auto-complete";

export class WorkflowRunFilter {
	key: string;
	options: Array<string>;
	example: string;
}

export class WorkflowRunFilterValue {
	key: string;
	value: string;
}

@Component({
	selector: 'app-projectv2-run-list',
	templateUrl: './run-list.html',
	styleUrls: ['./run-list.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectV2WorkflowRunListComponent implements OnInit, AfterViewInit {
	@ViewChild('filterInput') filterInput: ElementRef;
	@ViewChild('filterInputDirective') filterInputDirective: NzAutocompleteTriggerDirective;

	loading = false;
	totalCount: number = 0;
	runs: Array<V2WorkflowRun> = [];
	project: Project;
	filtersValue: Array<WorkflowRunFilterValue> = [];
	filters: Array<WorkflowRunFilter> = [];
	availableFilters: Array<WorkflowRunFilter> = [];
	filterText: string = '';
	selectedFilter: WorkflowRunFilter = null;
	textFilters = [];
	cursorTextFilterPosition: number = 0;

	constructor(
		private _http: HttpClient,
		private _messageService: NzMessageService,
		private _cd: ChangeDetectorRef,
		private _store: Store
	) {
		this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
	}

	ngOnInit(): void {
		this.loadFilters();
		this.search();
	}

	ngAfterViewInit(): void {
		const callback = this.filterInputDirective.handleKeydown.bind(this.filterInputDirective);
		this.filterInputDirective.handleKeydown = (event: KeyboardEvent): void => {
			if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
				this.computeAvailableFilters(this.filterText);
			}
			if ((event.key === 'ArrowLeft' || event.key === 'ArrowRight' || event.key === 'ArrowDown') && !this.filterInputDirective.panelOpen) {
				this.filterInputDirective.openPanel();
				return;
			}
			if (event.key === 'Enter') {
				if (this.filterInputDirective.activeOption && this.filterInputDirective.activeOption.nzValue !== this.filterText) {
					if (this.filterInputDirective.activeOption.nzValue.endsWith(':')) {
						event.preventDefault();
					}
					this.onFilterChange(this.filterInputDirective.activeOption.nzValue);
					return;
				} else if (this.filterInputDirective.activeOption) {
					this.submitForm();
				}
			}
			if (event.key === 'Escape') {
				this.filterInputDirective.closePanel();
				return;
			}
			callback(event);
		};
	}

	submitForm(): void {
		this.search();
	}

	onClickInput(): void {
		this.computeAvailableFilters(this.filterText);
		if (!this.filterInputDirective.panelOpen) {
			this.filterInputDirective.openPanel();
		}
	}

	async loadFilters() {
		this.loading = true;
		this._cd.markForCheck();

		try {
			this.filters = await lastValueFrom(this._http.get<Array<WorkflowRunFilter>>(`/v2/project/${this.project.key}/run/filter`));
			this.computeAvailableFilters(this.filterText);
		} catch (e) {
			this._messageService.error(`Unable to list workflow runs filters: ${e?.error?.error}`, { nzDuration: 2000 });
		}

		this.loading = false;
		this._cd.markForCheck();
	}

	async search(offset?: number) {
		this.loading = true;
		this._cd.markForCheck();

		let mFilters = {};
		this.filterText.split(' ').forEach(f => {
			const s = f.split(':');
			if (s.length === 2) {
				if (!mFilters[s[0]]) {
					mFilters[s[0]] = [];
				}
				mFilters[s[0]].push(s[1]);
			}
		});

		let params = {
			...mFilters,
			offset: offset ?? 0,
			limit: 20
		};

		try {
			const res = await lastValueFrom(this._http.get(`/v2/project/${this.project.key}/run/search`, { params, observe: 'response' })
				.pipe(map(res => {
					let headers: HttpHeaders = res.headers;
					return {
						totalCount: parseInt(headers.get('X-Total-Count'), 10),
						runs: res.body as Array<V2WorkflowRun>
					};
				})));
			this.totalCount = res.totalCount;
			this.runs = res.runs;
		} catch (e) {
			this._messageService.error(`Unable to list workflow runs: ${e?.error?.error}`, { nzDuration: 2000 });
		}

		this.loading = false;
		this._cd.markForCheck();
	}

	edit(item: any): void {
		this._messageService.success(item.email);
	}

	pageIndexChange(index: number): void {
		this.search((index - 1) * 20);
	}

	onFilterChange(originalText: string): void {
		this.computeAvailableFilters(originalText);
		this.filterText = originalText;
		this._cd.markForCheck();
	}

	computeAvailableFilters(originalText: string): void {
		// Get and adjust cursor position
		const originalCursorPosition = this.filterInput.nativeElement.selectionStart;
		this.textFilters = originalText.split(' ');
		// Retrieve the active filter in the text
		this.cursorTextFilterPosition = 0;
		let count = 0;
		this.textFilters.forEach((filter, idx) => {
			if (idx > 0) { count++ }; // Add +1 that match the space
			if (count <= originalCursorPosition && originalCursorPosition <= count + filter.length) {
				this.cursorTextFilterPosition = idx;
			}
			count += filter.length;
		});

		const splitted = this.textFilters[this.cursorTextFilterPosition].split(':');
		if (splitted.length === 2) {
			// Search for existing filter key to show options
			this.selectedFilter = Object.assign({}, this.filters.find(f => f.key === splitted[0]));
			if (this.selectedFilter) {
				this.selectedFilter.options = this.selectedFilter.options.filter(o => splitted[1] === '' || o.startsWith(splitted[1]));
			}
			this.availableFilters = [];
		} else {
			this.availableFilters = [].concat(this.filters);
			this.selectedFilter = null;
		}
	}

	computeFilterValue(filter: WorkflowRunFilter, option?: string): string {
		const textFilters = [].concat(this.textFilters);
		textFilters[this.cursorTextFilterPosition] = filter.key + ':' + (option ?? '');
		return textFilters.join(' ');
	}
}