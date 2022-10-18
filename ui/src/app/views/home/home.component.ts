import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { NavbarProjectData } from 'app/model/navbar.model';
import { Subscription } from 'rxjs';
import { TimelineFilter } from 'app/model/timeline.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { TimelineStore } from 'app/service/timeline/timeline.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-home',
    templateUrl: './home.html',
    styleUrls: ['./home.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class HomeComponent implements OnInit, OnDestroy {

    favorites: Array<NavbarProjectData> = [];
    loading = true;

    filter: TimelineFilter;
    filterSub: Subscription;

    _navbarSub: Subscription;

    constructor(
        private _navbarService: NavbarService,
        private _store: Store,
        private _timelineStore: TimelineStore,
        private _cd: ChangeDetectorRef
    ) {
        this.filter = new TimelineFilter();
        this._navbarSub = this._navbarService.getObservable()
            .subscribe((data) => {
                if (Array.isArray(data)) {
                    this.favorites = data.filter((fav) => fav.favorite);
                }
                this.loading = false;
                this._cd.markForCheck();
            });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.filterSub = this._timelineStore.getFilter().subscribe(f => {
            this.filter = f;
            this._cd.markForCheck();
        });
    }
}
