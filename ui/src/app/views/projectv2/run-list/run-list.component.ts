import { HttpClient, HttpHeaders, HttpParams } from "@angular/common/http";
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom, map, Subscription } from "rxjs";
import { Project } from "app/model/project.model";
import { Store } from "@ngxs/store";
import { ActivatedRoute, Router } from "@angular/router";
import * as actionPreferences from 'app/store/preferences.action';
import { PreferencesState } from "app/store/preferences.state";
import { NzPopconfirmDirective } from "ng-zorro-antd/popconfirm";
import { V2WorkflowRun } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { NzDrawerService } from "ng-zorro-antd/drawer";
import { ProjectV2RunStartComponent } from "../run-start/run-start.component";
import { EventV2Service } from "app/event-v2.service";
import { WebsocketV2Filter, WebsocketV2FilterType } from "app/model/websocket-v2";
import { EventV2State } from "app/store/event-v2.state";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { EventV2Type } from "app/model/event-v2.model";
import { animate, keyframes, state, style, transition, trigger } from "@angular/animations";
import { ErrorUtils } from "app/shared/error.utils";
import { ProjectV2State } from "app/store/project-v2.state";
import { Filter, FilterText } from "../../../shared/input/input-filter.component";
import { Clipboard } from '@angular/cdk/clipboard';
import { SearchService } from "app/service/search.service";
import { RUN_SUMMARY_HEADERS } from "app/service/workflowv2/workflow.service";
import { SearchResultType } from "app/model/search.model";
import { DisplaySearchResult } from "app/views/search/search.component";
import { WorkflowNameComponent } from "app/shared/workflow-name/workflow-name.component";

@Component({
	standalone: false,
	selector: 'app-projectv2-run-list',
	templateUrl: './run-list.html',
	styleUrls: ['./run-list.scss'],
	animations: [
		trigger('appendToList', [
			state('active', style({
				opacity: 1
			})),
			state('append', style({
				opacity: 1
			})),
			transition('append => active', animate('0ms')),
			transition('active => append', animate('1000ms', keyframes([
				style({ opacity: 1 }),
				style({ opacity: 0.5 }),
				style({ opacity: 1 })
			])))
		])
	],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RunListComponent implements OnInit, OnDestroy {
	static PANEL_KEY = 'project-v2-run-list-sidebar';
	static DEFAULT_SORT = 'started:desc';
	static DEFAULT_PAGESIZE = 20;
	static MAX_MATCHING_WORKFLOWS = 5;

	@ViewChild('saveSearchButton') saveSearchButton: NzPopconfirmDirective;

	loading = false;
	totalCount: number = 0;
	runs: Array<V2WorkflowRun> = [];
	project: Project;
	filters: Array<Filter> = [];
	filterText: string = '';
	previousFilterText: string = null;
	pageIndex: number = 1;
	panelSize: number | string;
	searchName: string = '';
	sort: string = ProjectV2RunListComponent.DEFAULT_SORT;
	eventV2Subscription: Subscription;
	animatedRuns: { [key: string]: boolean } = {};
	matchingWorkflows: Array<DisplaySearchResult> = [];

	private _http = inject(HttpClient);
	private _searchService = inject(SearchService);
	private _messageService = inject(NzMessageService);
	private _cd = inject(ChangeDetectorRef);
	private _store = inject(Store);
	private _router = inject(Router);
	private _activatedRoute = inject(ActivatedRoute);
	private _drawerService = inject(NzDrawerService);
	private _eventV2Service = inject(EventV2Service);
	private _clipboard = inject(Clipboard);

	constructor() {
		this.project = this._store.selectSnapshot(ProjectV2State.current);
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunListComponent.PANEL_KEY));
		this.loadFilters();
		this._activatedRoute.queryParams.subscribe(values => {
			this.filterText = FilterText.fromQueryParams(values, ['page', 'sort']);
			this.pageIndex = values['page'] ?? 1;
			this.sort = values['sort'] ?? ProjectV2RunListComponent.DEFAULT_SORT;
			this.search();
		});
		this.eventV2Subscription = this._store.select(EventV2State.last).subscribe((event) => {
			if (!event || [EventV2Type.EventRunCrafted, EventV2Type.EventRunBuilding, EventV2Type.EventRunEnded, EventV2Type.EventRunRestart].indexOf(event.type) === -1) { return; }
			const idx = this.runs.findIndex(run => run.id === event.workflow_run_id);
			delete (this.animatedRuns[event.payload.id]);
			this._cd.detectChanges();
			if (idx !== -1) {
				this.runs[idx] = event.payload;
			} else {
				this.runs = [event.payload].concat(...this.runs);
				if (this.runs.length > ProjectV2RunListComponent.DEFAULT_PAGESIZE) {
					this.runs.pop();
				}
				// The run pushed by the websocket matches the current search, so the empty result
				// state and the workflows it was suggesting do not stand anymore.
				this.totalCount++;
				this.matchingWorkflows = [];
			}
			this.animatedRuns[event.payload.id] = true;
			this._cd.markForCheck();
		});
	}

	changeFilter(v: string): void {
		this.filterText = v;
	}

	submitForm(): void {
		this.saveSearchInQueryParams();
	}

	async loadFilters() {
		this.loading = true;
		this._cd.markForCheck();

		try {
			this.filters = await lastValueFrom(this._http.get<Array<Filter>>(`/v2/project/${this.project.key}/run/filter`));
		} catch (e) {
			this._messageService.error(`Unable to list workflow runs filters: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
		}

		this.loading = false;
		this._cd.markForCheck();
	}

	async search() {
		this.loading = true;
		this._cd.markForCheck();

		this.previousFilterText = this.filterText;

		let params = new HttpParams();
		params = params.appendAll({
			...FilterText.toSearchParams(this.filterText),
			offset: this.pageIndex ? (this.pageIndex - 1) * ProjectV2RunListComponent.DEFAULT_PAGESIZE : 0,
			limit: ProjectV2RunListComponent.DEFAULT_PAGESIZE
		});
		if (this.sort !== ProjectV2RunListComponent.DEFAULT_SORT) {
			params = params.append('sort', this.sort);
		}

		this._eventV2Service.updateFilter(<WebsocketV2Filter>{
			type: WebsocketV2FilterType.PROJECT_RUNS,
			project_key: this.project.key,
			project_runs_params: params.toString()
		});

		try {
			const res = await lastValueFrom(this._http.get(`/v2/project/${this.project.key}/run`, {
				params,
				headers: RUN_SUMMARY_HEADERS,
				observe: 'response'
			})
				.pipe(map(res => {
					let headers: HttpHeaders = res.headers;
					return {
						totalCount: parseInt(headers.get('X-Total-Count'), 10),
						runs: res.body as Array<V2WorkflowRun>
					};
				})));
			this.totalCount = res.totalCount;
			this.runs = res.runs;
			await this.loadMatchingWorkflows();
		} catch (e) {
			this._messageService.error(`Unable to list workflow runs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
		}

		this.loading = false;
		this._cd.markForCheck();
	}

	// Without any run to show, look for the workflows the search was aiming at, so that a workflow
	// that exists but never ran can still be reached instead of ending on an empty list.
	async loadMatchingWorkflows(): Promise<void> {
		this.matchingWorkflows = [];
		if (this.totalCount > 0) {
			return;
		}

		const { filters, query } = FilterText.parse(this.filterText);
		const workflows = filters['workflow'] ?? [];
		const terms = workflows.length === 1 ? workflows : (workflows.length === 0 ? query : []);
		if (terms.length === 0) {
			return;
		}

		try {
			const res = await lastValueFrom(this._searchService.search({
				project: this.project.key,
				type: SearchResultType.Workflow,
				query: terms.join(' ')
			}, 0, ProjectV2RunListComponent.MAX_MATCHING_WORKFLOWS));
			this.matchingWorkflows = res.results
				.filter(r => r.type === SearchResultType.Workflow)
				.map(r => new DisplaySearchResult(r));
		} catch (e) {
			// Only a hint, a failure here must not hide the empty result state.
		}
	}

	workflowPath(result: DisplaySearchResult): string {
		return result.runLink.params['workflow'];
	}

	// The workflow name is the last segment, what precedes it is its vcs/repository scope. Same
	// split and same ref shortening as the source panel of a run, to read them alike.
	workflowScope(result: DisplaySearchResult): string {
		const path = this.workflowPath(result);
		const separator = path.lastIndexOf('/');
		return separator === -1 ? '' : path.substring(0, separator);
	}

	// A workflow can be defined on several refs, only the first one is shown inline with a counter
	// for the others, the tooltip carrying the full list.
	workflowRef(result: DisplaySearchResult): string {
		const refs = ProjectV2RunListComponent.shortRefs(result);
		if (refs.length === 0) {
			return null;
		}
		return refs.length === 1 ? refs[0] : `${refs[0]} +${refs.length - 1}`;
	}

	workflowTitleTooltip(result: DisplaySearchResult): string {
		const refs = ProjectV2RunListComponent.shortRefs(result);
		return this.workflowPath(result) + (refs.length > 0 ? ` @${refs.join(', ')}` : '');
	}

	private static shortRefs(result: DisplaySearchResult): Array<string> {
		return (result.result.variants ?? []).map(r => WorkflowNameComponent.shortRef(r));
	}

	saveSearchInQueryParams() {
		let queryParams = FilterText.toQueryParams(this.filterText);
		if (this.pageIndex > 1) {
			queryParams['page'] = this.pageIndex;
		}
		if (this.sort !== ProjectV2RunListComponent.DEFAULT_SORT) {
			queryParams['sort'] = this.sort;
		}

		this._router.navigate([], {
			relativeTo: this._activatedRoute,
			queryParams
		});
	}

	edit(item: any): void {
		this._messageService.success(item.email);
	}

	pageIndexChange(index: number): void {
		this.pageIndex = index;
		this._cd.markForCheck();
		this.saveSearchInQueryParams();
	}

	panelStartResize(): void {
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
	}

	panelEndResize(size: string): void {
		this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: ProjectV2RunListComponent.PANEL_KEY, size: size }));
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
			sort: this.sort !== ProjectV2RunListComponent.DEFAULT_SORT ? this.sort : null
		}));
		this.searchName = '';
	}

	onSearchNameChange(name: string): void {
		this.searchName = name;
	}

	refresh(e: Event = null): void {
		if (this.filterText !== this.previousFilterText) {
			return;
		}
		this.search();
		if (e) {
			e.preventDefault();
			e.stopPropagation();
		}
	}

	onSortChange(sort: string): void {
		this.sort = sort;
		this._cd.markForCheck();
		this.saveSearchInQueryParams();
	}

	openRunStartDrawer(workflow: string = null): void {
		// A workflow picked from an empty result set must not inherit the filters that found nothing.
		const mFilters = workflow ? { workflow: [workflow] } : FilterText.parse(this.filterText).filters;
		const drawerRef = this._drawerService.create<ProjectV2RunStartComponent, { value: string }, string>({
			nzTitle: 'Start new Workflow Run',
			nzContent: ProjectV2RunStartComponent,
			nzContentParams: {
				params: {
					workflow_repository: mFilters['workflow_repository'] ? mFilters['workflow_repository'][0] : null,
					repository: mFilters['repository'] ? mFilters['repository'][0] : null,
					workflow_ref: mFilters['workflow_ref'] ? mFilters['workflow_ref'][0] : null,
					ref: mFilters['ref'] ? mFilters['ref'][0] : null,
					workflow: mFilters['workflow'] ? mFilters['workflow'][0] : null
				}
			},
			nzSize: 'large',
			nzBodyStyle: { 'padding': '0' }
		});
		drawerRef.afterClose.subscribe(data => { });
	}

	generateAnnotationQueryParams(annotation: { key: string, value: string }): any {
		let queryParams = {};
		queryParams[annotation.key] = annotation.value;
		return queryParams;
	}

	async clickDeleteRun(runID: string) {
		try {
			await lastValueFrom(this._http.delete(`/v2/project/${this.project.key}/run/${runID}`));
			this.refresh();
		} catch (e) {
			this._messageService.error(`Unable to delete workflow run: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
		}
	}

	trackRunElement(index: number, run: V2WorkflowRun): any {
		return run.id;
	}

	// Opening a run by clicking anywhere on its line is a convenience for the mouse, the keyboard
	// reaches it through the run link of the line. Clicks meant for another control of the line, a
	// modified click or the end of a text selection are left alone.
	clickRunLine(event: MouseEvent, run: V2WorkflowRun): void {
		if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
			return;
		}
		if ((event.target as HTMLElement).closest('a, button, nz-tag, [role="button"]')) {
			return;
		}
		if (window.getSelection()?.toString()) {
			return;
		}
		this._router.navigate(['/project', run.project_key, 'run', run.id]);
	}

	onMouseEnterRun(id: string): void {
		delete this.animatedRuns[id];
		this._cd.markForCheck();
	}

	confirmCopyAnnotationValue(event: any, value: string) {
		event.stopPropagation();
		event.preventDefault();
		this._clipboard.copy(value);
		this._messageService.success('Annotation value copied!');
	}
}
