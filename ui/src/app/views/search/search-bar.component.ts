import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { Router } from "@angular/router";
import { SearchResult, SearchResultType } from "app/model/search.model";
import { SearchService } from "app/service/search.service";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import Debounce from "app/shared/decorator/debounce";
import { ErrorUtils } from "app/shared/error.utils";
import { Filter, InputFilterComponent, Suggestion } from "app/shared/input/input-filter.component";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom } from "rxjs";
import { DisplaySearchResult } from "./search.component";

@Component({
	selector: 'app-search-bar',
	templateUrl: './search-bar.html',
	styleUrls: ['./search-bar.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class SearchBarComponent implements OnInit, OnDestroy {
	@ViewChild('searchBar') searchBar: InputFilterComponent<Suggestion<SearchResult>>;

	searchFilterText: string = '';
	searchFilters: Array<Filter> = [];
	searchSuggestions: Array<Suggestion<DisplaySearchResult>> = [];
	loading: boolean;

	constructor(
		private _searchService: SearchService,
		private _messageService: NzMessageService,
		private _cd: ChangeDetectorRef,
		private _router: Router
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.loadFilters();
	}

	selectSuggestion(value: DisplaySearchResult): void {
		this._router.navigate(value.defaultLink.path, { queryParams: value.defaultLink.params });
	}

	submitSearch(): void {
		let mFilters = {};
		this.searchFilterText.split(' ').forEach(f => {
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

		this._router.navigate(['/search'], {
			queryParams: { ...mFilters }
		});
	}

	searchChange(v: string) {
		this.searchFilterText = v;
		this.search();
	}

	@Debounce(300)
	async search() {
		this.loading = true;
		this._cd.markForCheck();

		let mFilters = {};
		this.searchFilterText.split(' ').forEach(f => {
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
			const res = await lastValueFrom(this._searchService.search(mFilters, 0, 10));
			this.searchSuggestions = res.results.map(r => ({
				key: r.id,
				label: `${r.label} - ${r.id}`,
				data: new DisplaySearchResult(r),
			}));
		} catch (e: any) {
			this._messageService.error(`Unable to search: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
		}
		this.loading = false;
		this._cd.markForCheck();
	}

	async loadFilters() {
		this.loading = true;
		this._cd.markForCheck();

		try {
			this.searchFilters = await lastValueFrom(this._searchService.getFilters());
		} catch (e) {
			this._messageService.error(`Unable to list search filters: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
		}

		this.loading = false;
		this._cd.markForCheck();
	}

	clickSuggestion(): void {
		this.searchBar.filterInputDirective.closePanel();
	}
}