import { ChangeDetectorRef, Component, OnInit } from "@angular/core";
import { ActivatedRoute, Router } from "@angular/router";
import { SearchResult, SearchResultType } from "app/model/search.model";
import { SearchService } from "app/service/search.service";
import Debounce from "app/shared/decorator/debounce";
import { ErrorUtils } from "app/shared/error.utils";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom } from "rxjs";

@Component({
	selector: 'app-search',
	templateUrl: './search.html',
	styleUrls: ['./search.scss']
})
export class SearchComponent implements OnInit {
	static DEFAULT_PAGESIZE = 20;

	results: Array<SearchResult> = [];
	loading = false;
	query: string = '';
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
		this._activatedRoute.queryParams.subscribe(values => {
			this.query = values['query'] ?? '';
			this.pageIndex = values['page'] ?? 1;
			this.search();
		});

		this.search();
	}

	@Debounce(200)
	queryChange(query: string) {
		this.query = query;
		this.saveSearchInQueryParams();
	}

	async search() {
		this.loading = true;
		this._cd.markForCheck();
		try {
			const res = await lastValueFrom(this._searchService.search(this.query,
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
		let queryParams = {};
		if (this.query.length > 0) {
			queryParams['query'] = this.query;
		}
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
			case SearchResultType.WorkflowV2:				
				const project = splitted.shift();
				const vcs = splitted.shift();
				const workflow = splitted.pop();
				const repository = splitted.join('/');
				return ['/project', project, 'explore', 'vcs', vcs, 'repository', repository, 'workflow', workflow];
			case SearchResultType.Workflow:
				return ['/project', splitted[0], 'workflow', splitted[1]];
			case SearchResultType.Project:
				return ['/project', res.id];
			default:
				return [];
		}
	}
}