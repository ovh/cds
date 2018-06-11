import {Component} from '@angular/core';
import {BroadcastService} from '../../../service/broadcast/broadcast.service';
import {Broadcast} from 'app/model/broadcast.model';
import {Subscription} from 'rxjs';
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

    constructor(private _broadcastService: BroadcastService) {
      this._broadcastSub = this._broadcastService.getBroadcastsListener()
        .subscribe((broadcasts) => {
            this.loading = false;
            this.recentBroadcasts = broadcasts.filter((br) => !br.read && !br.archived);
            this.oldBroadcasts = broadcasts.filter((br) => br.read || br.archived);
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
      this._broadcastService.markAsRead(id)
        .pipe(finalize(() => this.loading = false))
        .subscribe();
    }
}
