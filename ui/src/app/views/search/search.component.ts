import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from "@angular/core";
import { ActivatedRoute, Router } from "@angular/router";
import { SearchResult, SearchResultType } from "app/model/search.model";
import { SearchService } from "app/service/search.service";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { ErrorUtils } from "app/shared/error.utils";
import { Filter } from "app/shared/input/input-filter.component";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom, Subscription } from "rxjs";

export class Link {
	path: Array<string>;
	params: { [key: string]: string };
}

export class DisplaySearchResult {
	result: SearchResult;
	unfold: boolean;
	defaultLink: Link;
	exploreLink: Link;
	runLink: Link;

	constructor(r: SearchResult) {
		this.result = r;
		this.generateLinks();
	}

	generateLinks(): void {
		const splitted = this.result.id.split('/');
		switch (this.result.type) {
			case SearchResultType.Workflow:
				const project = splitted.shift();
				const workflow_path = splitted.join('/');
				const vcs = splitted.shift();
				const name = splitted.pop();
				const repository = splitted.join('/');
				this.exploreLink = {
					path: ['/project', project, 'explore', 'vcs', vcs, 'repository', repository, 'workflow', name],
					params: {}
				}
				this.runLink = {
					path: ['/project', project, 'run'],
					params: { workflow: workflow_path }
				};
				this.defaultLink = this.runLink;
				break;
			case SearchResultType.WorkflowLegacy:
				this.defaultLink = {
					path: ['/project', splitted[0], 'workflow', splitted[1]],
					params: {}
				};
				break;
			case SearchResultType.Project:
				this.exploreLink = {
					path: ['/project', this.result.id, 'explore'],
					params: {}
				}
				this.runLink = {
					path: ['/project', this.result.id, 'run'],
					params: {}
				};
				this.defaultLink = {
					path: ['/project', this.result.id],
					params: {}
				}
				break;
		}
	}

	generateVariantRunLink(variant?: string): Link {
		switch (this.result.type) {
			case SearchResultType.Workflow:
				return {
					path: this.runLink.path,
					params: { ...this.runLink.params, ref: variant }
				};
			default:
				return null;
		}
	}
}

@Component({
	selector: 'app-search',
	templateUrl: './search.html',
	styleUrls: ['./search.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class SearchComponent implements OnInit, OnDestroy {
	static DEFAULT_PAGESIZE = 20;

	queryParamsSub: Subscription;
	filters: Array<Filter> = [];
	results: Array<DisplaySearchResult> = [];
	loading: boolean;
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

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.loadFilters();
		this.queryParamsSub = this._activatedRoute.queryParams.subscribe(values => {
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
			this.results = res.results.map(r => new DisplaySearchResult(r));
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
			queryParams
		});
	}

	pageIndexChange(index: number): void {
		this.pageIndex = index;
		this._cd.markForCheck();
		this.saveSearchInQueryParams();
	}

	unfoldItem(id: string): void {
		for (let i = 0; i < this.results.length; i++) {
			if (this.results[i].result.id === id) {
				this.results[i].unfold = true;
				break;
			}
		}
		this._cd.markForCheck();
	}
}