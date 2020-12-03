import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { Broadcast } from 'app/model/broadcast.model';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { BroadcastStore } from '../../../service/broadcast/broadcast.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-broadcast-list',
    templateUrl: './broadcast.list.component.html',
    styleUrls: ['./broadcast.list.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class BroadcastListComponent implements OnDestroy {
    recentBroadcasts: Array<Broadcast> = [];
    oldBroadcasts: Array<Broadcast> = [];
    filteredBroadcasts: Array<Broadcast> = [];
    loading = true;

    recentView = true;

    _broadcastSub: Subscription;

    set filter(filter: string) {
        let filterLower = filter.toLowerCase();
        let broadcasts = this.recentView ? this.recentBroadcasts : this.oldBroadcasts;
        this.filteredBroadcasts = broadcasts.filter((br) => br.title.toLowerCase().indexOf(filterLower) !== -1 || br.level === filter);
    }

    constructor(private _broadcastStore: BroadcastStore, private _cd: ChangeDetectorRef) {
        this._broadcastSub = this._broadcastStore.getBroadcasts()
            .subscribe((broadcasts) => {
                this.recentBroadcasts = broadcasts.valueSeq().toArray().filter((br) => !br.read && !br.archived)
                    .sort((a, b) => (new Date(b.updated)).getTime() - (new Date(a.updated)).getTime());
                this.oldBroadcasts = broadcasts.valueSeq().toArray().filter((br) => br.read || br.archived)
                    .sort((a, b) => (new Date(b.updated)).getTime() - (new Date(a.updated)).getTime());
                this.filteredBroadcasts = this.recentBroadcasts;
                this.loading = false;
                this._cd.markForCheck();
            });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    switchToRecentView(recent: boolean) {
        let filterLower = '';
        if (this.filter) {
            filterLower = this.filter.toLowerCase();
        }
        this.recentView = recent;
        if (recent) {
            this.filteredBroadcasts = this.recentBroadcasts.filter((br) => br.title.toLowerCase().indexOf(filterLower) !== -1 || br.level === this.filter);
        } else {
            this.filteredBroadcasts = this.oldBroadcasts.filter((br) => br.title.toLowerCase().indexOf(filterLower) !== -1 || br.level === this.filter);
        }
    }

    markAsRead(id: number) {
        this.loading = true;
        this._broadcastStore.markAsRead(id)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe();
    }
}
