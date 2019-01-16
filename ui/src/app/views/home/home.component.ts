import {Component, OnInit} from '@angular/core';
import {Broadcast} from 'app/model/broadcast.model';
import {NavbarProjectData} from 'app/model/navbar.model';
import {Subscription} from 'rxjs';
import {TimelineFilter} from '../../model/timeline.model';
import {User} from '../../model/user.model';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {BroadcastStore} from '../../service/broadcast/broadcast.store';
import {NavbarService} from '../../service/navbar/navbar.service';
import {TimelineStore} from '../../service/timeline/timeline.store';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-home',
    templateUrl: './home.html',
    styleUrls: ['./home.scss']
})
@AutoUnsubscribe()
export class HomeComponent implements OnInit {

    selectedTab = 'heatmap';
    favorites: Array<NavbarProjectData> = [];
    broadcasts: Array<Broadcast> = [];
    loading = true;
    loadingBroadcasts = true;
    user: User;

    filter: TimelineFilter;
    filterSub: Subscription;

    _navbarSub: Subscription;
    _broadcastSub: Subscription;

    constructor(
      private _navbarService: NavbarService,
      private _broadcastService: BroadcastStore,
      private _authStore: AuthentificationStore,
      private _timelineStore: TimelineStore
    ) {
        this.user = this._authStore.getUser();
        this.filter = new TimelineFilter();
        this._navbarSub = this._navbarService.getData(true)
            .subscribe((data) => {
                this.loading = false;
                if (Array.isArray(data)) {
                    this.favorites = data.filter((fav) => fav.favorite);
                }
            });

        this._broadcastSub = this._broadcastService.getBroadcasts()
            .subscribe((broadcasts) => {
                this.loadingBroadcasts = false;
                if (broadcasts) {
                    this.broadcasts = broadcasts.toArray().filter((br) => !br.read && !br.archived).slice(0, 5);
                }
            }, () => this.loadingBroadcasts = false);
    }

    ngOnInit() {
        this.filterSub = this._timelineStore.getFilter().subscribe(f => {
            this.filter = f;
        });
    }

    selectTab(t: string): void {
        this.selectedTab = t;
    }

    markAsRead(id: number) {
        for (let i = 0; i < this.broadcasts.length; i++) {
            if (this.broadcasts[i].id === id) {
                this.broadcasts[i].updating = true;
            }
        }
        this._broadcastService.markAsRead(id)
            .subscribe();
    }
}
