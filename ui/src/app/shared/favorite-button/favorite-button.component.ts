import { Component, ChangeDetectionStrategy, ChangeDetectorRef, Input, OnInit, OnDestroy } from "@angular/core";
import { Store } from "@ngxs/store";
import { Bookmark, BookmarkType } from "app/model/bookmark.model";
import { Subscription } from "rxjs";
import { AutoUnsubscribe } from "../decorator/autoUnsubscribe";
import { BookmarkCreate, BookmarkDelete, BookmarkState } from "app/store/bookmark.state";

@Component({
    selector: 'app-favorite-button',
    templateUrl: './favorite-button.component.html',
    styleUrls: ['./favorite-button.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class FavoriteButtonComponent implements OnInit, OnDestroy {
    @Input() type: BookmarkType;
    @Input() id: string;
    loading: boolean;
    bookmark: Bookmark;
    sub: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.sub = this._store.select(BookmarkState.state).subscribe((s) => {
            this.loading = s.loading === (this.type + this.id);
            this.bookmark = s.all.find((b) => b.type === this.type && b.id === this.id);
            this._cd.markForCheck();
        });
    }

    click(): void {
        if (this.bookmark) {
            this._store.dispatch(new BookmarkDelete(this.bookmark));
        } else {
            this._store.dispatch(new BookmarkCreate(<Bookmark>{
                type: this.type,
                id: this.id
            }));
        }
    }
}
