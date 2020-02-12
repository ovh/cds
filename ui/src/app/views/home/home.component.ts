import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { Broadcast } from 'app/model/broadcast.model';
import { NavbarProjectData } from 'app/model/navbar.model';
import { AuthenticationState } from 'app/store/authentication.state';
import { Subscription } from 'rxjs';
import { TimelineFilter } from '../../model/timeline.model';
import { AuthentifiedUser } from '../../model/user.model';
import { BroadcastStore } from '../../service/broadcast/broadcast.store';
import { NavbarService } from '../../service/navbar/navbar.service';
import { TimelineStore } from '../../service/timeline/timeline.store';
import { AutoUnsubscribe } from '../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-home',
    templateUrl: './home.html',
    styleUrls: ['./home.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class HomeComponent implements OnInit {

    selectedTab = 'heatmap';
    favorites: Array<NavbarProjectData> = [];
    broadcasts: Array<Broadcast> = [];
    loading = true;
    loadingBroadcasts = true;
    user: AuthentifiedUser;

    filter: TimelineFilter;
    filterSub: Subscription;

    _navbarSub: Subscription;
    _broadcastSub: Subscription;

    constructor(
        private _navbarService: NavbarService,
        private _broadcastService: BroadcastStore,
        private _store: Store,
        private _timelineStore: TimelineStore,
        private _cd: ChangeDetectorRef
    ) {
        this.user = this._store.selectSnapshot(AuthenticationState.user);
        this.filter = new TimelineFilter();
        this._navbarSub = this._navbarService.getObservable()
            .subscribe((data) => {
                if (Array.isArray(data)) {
                    this.favorites = data.filter((fav) => fav.favorite);
                }
                this.loading = false;
                this._cd.markForCheck();
            });

        this._broadcastSub = this._broadcastService.getBroadcasts()
            .subscribe((broadcasts) => {
                this.loadingBroadcasts = false;
                if (broadcasts) {
                    this.broadcasts = broadcasts.valueSeq().toArray().filter((br) => !br.read && !br.archived).slice(0, 5);
                }
                this._cd.markForCheck();
            }, () => this.loadingBroadcasts = false);
    }

    ngOnInit() {
        this.filterSub = this._timelineStore.getFilter().subscribe(f => {
            this.filter = f;
            this._cd.markForCheck();
        });
    }

    selectTab(t: string): void {
        this.selectedTab = t;
    }
}
