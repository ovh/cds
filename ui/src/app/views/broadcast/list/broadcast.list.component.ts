import {Component} from '@angular/core';
import {BroadcastStore} from '../../../service/broadcast/broadcast.store';
import {Broadcast} from 'app/model/broadcast.model';
import {Subscription} from 'rxjs/Subscription';
import {finalize} from 'rxjs/operators';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-broadcast-list',
    templateUrl: './broadcast.list.component.html',
    styleUrls: ['./broadcast.list.component.scss']
})
@AutoUnsubscribe()
export class BroadcastListComponent {

    recentBroadcasts: Array<Broadcast> = [];
    oldBroadcasts: Array<Broadcast> = [];
    filteredBroadcasts: Array<Broadcast> = [];
    loading = true;

    recentView = true;

    _broadcastSub: Subscription;

    set filter(filter: string) {
        let filterLower = filter.toLowerCase();
        let broadcasts = this.recentView ? this.recentBroadcasts : this.oldBroadcasts;
        this.filteredBroadcasts = broadcasts.filter((br) => {
          return br.title.toLowerCase().indexOf(filterLower) !== -1 || br.level === filter;
        });
    }

    constructor(private _broadcastStore: BroadcastStore) {
      this._broadcastSub = this._broadcastStore.getBroadcasts()
        .subscribe((broadcasts) => {
            this.loading = false;
            this.recentBroadcasts = broadcasts.toArray().filter((br) => !br.read && !br.archived);
            this.oldBroadcasts = broadcasts.toArray().filter((br) => br.read || br.archived);
            this.filteredBroadcasts = this.recentBroadcasts;
        }, () => this.loading = false);
    }

    switchToRecentView(recent: boolean) {
        let filterLower = '';
        if (this.filter) {
            filterLower = this.filter.toLowerCase();
        }
        this.recentView = recent;
        if (recent) {
            this.filteredBroadcasts = this.recentBroadcasts.filter((br) => {
              return br.title.toLowerCase().indexOf(filterLower) !== -1 || br.level === this.filter;
            });
        } else {
            this.filteredBroadcasts = this.oldBroadcasts.filter((br) => {
              return br.title.toLowerCase().indexOf(filterLower) !== -1 || br.level === this.filter;
            });
        }
    }

    markAsRead(id: number) {
      this.loading = true;
      this._broadcastStore.markAsRead(id)
        .pipe(finalize(() => this.loading = false))
        .subscribe();
    }
}
