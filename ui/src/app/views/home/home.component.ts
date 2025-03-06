import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from "@angular/core";
import { Store } from "@ngxs/store";
import { Bookmark, BookmarkType } from "app/model/bookmark.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { BookmarkDelete, BookmarkLoad, BookmarkState } from "app/store/bookmark.state";
import { Subscription } from "rxjs";

@Component({
	selector: 'app-home',
	templateUrl: './home.html',
	styleUrls: ['./home.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class HomeComponent implements OnInit, OnDestroy {
	
	bookmarks: Array<Bookmark> = [];
	bookmarksSubscription: Subscription;
	recentItems: Array<any> = [];
	loading: boolean;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.bookmarksSubscription = this._store.select(BookmarkState.state).subscribe((state) => {
			this.loading = !!state.loading;
			this.bookmarks = state.all;
			this._cd.markForCheck();
		});
		this._store.dispatch(new BookmarkLoad());
	}

	generateBookmarkLink(b: Bookmark): Array<string> {
		const splitted = b.id.split('/');
		switch (b.type) {
			case BookmarkType.Workflow:
				const project = splitted.shift();
				return ['/project', project, 'run'];
			case BookmarkType.WorkflowLegacy:
				return ['/project', splitted[0], 'workflow', splitted[1]];
			case BookmarkType.Project:
				return ['/project', b.id];
			default:
				return [];
		}
	}

	generateBookmarkQueryParams(b: Bookmark, variant?: string): any {
		const splitted = b.id.split('/');
		switch (b.type) {
			case BookmarkType.Workflow:
				splitted.shift();
				const workflow_path = splitted.join('/');
				let params = { workflow: workflow_path };
				if (variant) {
					params['ref'] = variant;
				}
				return params;
			default:
				return {};
		}
	}

	async deleteBookmark(e: Event, b: Bookmark) {
		e.preventDefault();
		e.stopPropagation();
		this._store.dispatch(new BookmarkDelete(b));
	}

}