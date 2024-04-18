import { HttpClient, HttpHeaders } from "@angular/common/http";
import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, OnInit, ViewChild } from "@angular/core";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom, map } from "rxjs";
import { Project } from "app/model/project.model";
import { Store } from "@ngxs/store";
import { ProjectState } from "app/store/project.state";
import { NzAutocompleteTriggerDirective } from "ng-zorro-antd/auto-complete";
import { ActivatedRoute, Router } from "@angular/router";
import * as actionPreferences from 'app/store/preferences.action';
import { PreferencesState } from "app/store/preferences.state";
import { NzPopconfirmDirective } from "ng-zorro-antd/popconfirm";
import { V2WorkflowRun } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";

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
	static PANEL_KEY = 'project-v2-run-list-sidebar';
	static DEFAULT_SORT = 'started:desc';

	@ViewChild('filterInput') filterInput: ElementRef;
	@ViewChild('filterInputDirective') filterInputDirective: NzAutocompleteTriggerDirective;
	@ViewChild('saveSearchButton') saveSearchButton: NzPopconfirmDirective;

	loading = false;
	totalCount: number = 0;
	runs: Array<V2WorkflowRun> = [];
	project: Project;
	filtersValue: Array<WorkflowRunFilterValue> = [];
	filters: Array<WorkflowRunFilter> = [];
	availableFilters: Array<WorkflowRunFilter> = [];
	filterText: string = '';
	previousFilterText: string = null;
	selectedFilter: WorkflowRunFilter = null;
	textFilters = [];
	cursorTextFilterPosition: number = 0;
	index: number = 1;
	panelSize: number | string;
	searchName: string = '';
	sort: string = ProjectV2WorkflowRunListComponent.DEFAULT_SORT;

	constructor(
		private _http: HttpClient,
		private _messageService: NzMessageService,
		private _cd: ChangeDetectorRef,
		private _store: Store,
		private _router: Router,
		private _activatedRoute: ActivatedRoute
	) {
		this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
	}

	ngOnInit(): void {
		this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2WorkflowRunListComponent.PANEL_KEY));
		this.loadFilters();
		this._activatedRoute.queryParams.subscribe(values => {
			this.filterText = Object.keys(values).filter(key => key !== 'page' && key !== 'sort').map(key => {
				return (!Array.isArray(values[key]) ? [values[key]] : values[key]).map(f => {
					return `${key}:${f}`;
				}).join(' ');
			}).join(' ');
			this.index = values['page'] ?? 1;
			this.sort = values['sort'] ?? ProjectV2WorkflowRunListComponent.DEFAULT_SORT;
			this.search();
		});
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
					this.onFilterTextChange(this.filterInputDirective.activeOption.nzValue);
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
		this.saveSearchInQueryParams();
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

	async search() {
		this.loading = true;
		this._cd.markForCheck();

		this.previousFilterText = this.filterText;

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
			offset: this.index ? (this.index - 1) * 20 : 0,
			limit: 20
		};
		if (this.sort !== ProjectV2WorkflowRunListComponent.DEFAULT_SORT) {
			params['sort'] = this.sort;
		}

		try {
			const res = await lastValueFrom(this._http.get(`/v2/project/${this.project.key}/run`, { params, observe: 'response' })
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

	saveSearchInQueryParams() {
		let mFilters = {};
		this.filterText.split(' ').forEach(f => {
			const s = f.split(':');
			if (s.length === 2 && s[1] !== '') {
				if (!mFilters[s[0]]) {
					mFilters[s[0]] = [];
				}
				mFilters[s[0]].push(s[1]);
			}
		});

		let queryParams = { ...mFilters };
		if (this.index > 1) {
			queryParams['page'] = this.index;
		}
		if (this.sort !== ProjectV2WorkflowRunListComponent.DEFAULT_SORT) {
			queryParams['sort'] = this.sort;
		}

		this._router.navigate([], {
			relativeTo: this._activatedRoute,
			queryParams,
			replaceUrl: true
		});
	}

	edit(item: any): void {
		this._messageService.success(item.email);
	}

	pageIndexChange(index: number): void {
		this.index = index;
		this._cd.markForCheck();
		this.saveSearchInQueryParams();
	}

	onFilterTextChange(originalText: string): void {
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
				this.selectedFilter.options = this.selectedFilter.options.filter(o => splitted[1] === '' || o.toLowerCase().indexOf(splitted[1].toLowerCase()) !== -1);
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

	panelStartResize(): void {
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
	}

	panelEndResize(size: string): void {
		this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: ProjectV2WorkflowRunListComponent.PANEL_KEY, size: size }));
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
	}

	submitSaveSearch(): void {
		this.confirmSaveSearch();
		this.saveSearchButton.hide();
	}

	confirmSaveSearch(): void {
		this._store.dispatch(new actionPreferences.SaveProjectWorkflowRunFilter({
			projectKey: this.project.key,
			name: this.searchName,
			value: this.filterText,
			sort: this.sort !== ProjectV2WorkflowRunListComponent.DEFAULT_SORT ? this.sort : null
		}));
		this.searchName = '';
	}

	onSearchNameChange(name: string): void {
		this.searchName = name;
	}

	refresh(e: Event): void {
		if (this.filterText !== this.previousFilterText) {
			return;
		}
		this.search();
		e.preventDefault();
		e.stopPropagation();
	}

	onSortChange(sort: string): void {
		this.sort = sort;
		this._cd.markForCheck();
		this.saveSearchInQueryParams();
	}

}