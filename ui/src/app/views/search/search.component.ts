import { ChangeDetectorRef, Component, OnInit } from "@angular/core";
import { ActivatedRoute, Router } from "@angular/router";
import { SearchResult, SearchResultType } from "app/model/search.model";
import { SearchService } from "app/service/search.service";
import { ErrorUtils } from "app/shared/error.utils";
import { Filter } from "app/shared/input/input-filter.component";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom } from "rxjs";

@Component({
	selector: 'app-search',
	templateUrl: './search.html',
	styleUrls: ['./search.scss']
})
export class SearchComponent implements OnInit {
	static DEFAULT_PAGESIZE = 20;

	filters: Array<Filter> = [];
	results: Array<SearchResult> = [];
	loading = false;
	filterText: string = '';
	totalCount: number = 0;
	pageIndex: number = 1;

	constructor(
		private _searchService: SearchService,
		private _messageService: NzMessageService,
		private _cd: ChangeDetectorRef,
		private _router: Router,
		private _activatedRoute: ActivatedRoute
	) { }

	ngOnInit(): void {
		this.loadFilters();
		this._activatedRoute.queryParams.subscribe(values => {
			this.filterText = Object.keys(values).filter(key => key !== 'page').map(key => {
				return (!Array.isArray(values[key]) ? [values[key]] : values[key]).map(f => {
					return key === 'query' ? f : `${key}:${f}`;
				}).join(' ');
			}).join(' ');
			this.pageIndex = values['page'] ?? 1;
			this.search();
		});
	}

	async loadFilters() {
		this.loading = true;
		this._cd.markForCheck();

		try {
			this.filters = await lastValueFrom(this._searchService.getFilters());
		} catch (e) {
			this._messageService.error(`Unable to list search filters: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
		}

		this.loading = false;
		this._cd.markForCheck();
	}

	filterChange(v: string) {
		this.filterText = v;
	}

	submitForm(): void {
		this.saveSearchInQueryParams();
	}

	async search() {
		this.loading = true;
		this._cd.markForCheck();

		let mFilters = {};
		this.filterText.split(' ').forEach(f => {
			const s = f.split(':');
			if (s.length === 2) {
				if (!mFilters[s[0]]) {
					mFilters[s[0]] = [];
				}
				mFilters[s[0]].push(decodeURI(s[1]));
			} else if (s.length === 1) {
				mFilters['query'] = f;
			}
		});

		try {
			const res = await lastValueFrom(this._searchService.search(mFilters,
				this.pageIndex ? (this.pageIndex - 1) * SearchComponent.DEFAULT_PAGESIZE : 0,
				SearchComponent.DEFAULT_PAGESIZE));
			this.totalCount = res.totalCount;
			this.results = res.results.map(r => ({
				...r,
				variants: r.variants ? r.variants.filter((v, i) => r.variants.indexOf(v) === i) : null
			}));
		} catch (e: any) {
			this._messageService.error(`Unable to search: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
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
			} else if (s.length === 1) {
				mFilters['query'] = f;
			}
		});

		let queryParams = { ...mFilters };
		if (this.pageIndex > 1) {
			queryParams['page'] = this.pageIndex;
		}

		this._router.navigate([], {
			relativeTo: this._activatedRoute,
			queryParams,
			replaceUrl: true
		});
	}

	pageIndexChange(index: number): void {
		this.pageIndex = index;
		this._cd.markForCheck();
		this.saveSearchInQueryParams();
	}

	generateResultLink(res: SearchResult): Array<string> {
		const splitted = res.id.split('/');
		switch (res.type) {
			case SearchResultType.Workflow:
				const project = splitted.shift();
				const vcs = splitted.shift();
				const workflow = splitted.pop();
				const repository = splitted.join('/');
				return ['/project', project, 'explore', 'vcs', vcs, 'repository', repository, 'workflow', workflow];
			case SearchResultType.WorkflowLegacy:
				return ['/project', splitted[0], 'workflow', splitted[1]];
			case SearchResultType.Project:
				return ['/project', res.id];
			default:
				return [];
		}
	}
}