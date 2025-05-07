import { HttpClient, HttpHeaders, HttpParams } from "@angular/common/http";
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from "@angular/core";
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
import { Filter } from "../../../shared/input/input-filter.component";
import { Clipboard } from '@angular/cdk/clipboard';
import { ToastService } from "app/shared/toast/ToastService";

@Component({
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

	constructor(
		private _http: HttpClient,
		private _messageService: NzMessageService,
		private _cd: ChangeDetectorRef,
		private _store: Store,
		private _router: Router,
		private _activatedRoute: ActivatedRoute,
		private _drawerService: NzDrawerService,
		private _eventV2Service: EventV2Service,
		private _clipboard: Clipboard,
		private _toast: ToastService
	) {
		this.project = this._store.selectSnapshot(ProjectV2State.current);
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunListComponent.PANEL_KEY));
		this.loadFilters();
		this._activatedRoute.queryParams.subscribe(values => {
			this.filterText = Object.keys(values).filter(key => key !== 'page' && key !== 'sort').map(key => {
				return (!Array.isArray(values[key]) ? [values[key]] : values[key]).map(f => {
					return `${key}:${f}`;
				}).join(' ');
			}).join(' ');
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

		let mFilters = {};
		this.filterText.split(' ').forEach(f => {
			const s = f.split(':');
			if (s.length === 2) {
				if (!mFilters[s[0]]) {
					mFilters[s[0]] = [];
				}
				mFilters[s[0]].push(decodeURI(s[1]));
			}
		});

		let params = new HttpParams();
		params = params.appendAll({
			...mFilters,
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
			this._messageService.error(`Unable to list workflow runs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
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

	openRunStartDrawer(): void {
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
		const drawerRef = this._drawerService.create<ProjectV2RunStartComponent, { value: string }, string>({
			nzTitle: 'Start new worklfow run',
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
			nzSize: 'large'
		});
		drawerRef.afterClose.subscribe(data => { });
	}

	generateAnnotationQueryParams(annotation: { key: string, value: string }): any {
		let queryParams = {};
		queryParams[annotation.key] = encodeURI(annotation.value);
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

	onMouseEnterRun(id: string): void {
		delete this.animatedRuns[id];
		this._cd.markForCheck();
	}

	confirmCopyAnnotationValue(event: any, value: string) {
		event.stopPropagation();
		event.preventDefault();
		this._clipboard.copy(value);
		this._toast.success('', 'Annotation value copied!');
	}
}