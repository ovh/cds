import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Broadcast } from 'app/model/broadcast.model';
import { Subscription } from 'rxjs';
import { BroadcastStore } from '../../../service/broadcast/broadcast.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-broadcast-details',
    templateUrl: './broadcast.details.component.html',
    styleUrls: ['./broadcast.details.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class BroadcastDetailsComponent implements OnDestroy {

    broadcast: Broadcast;
    loading = true;

    _broadcastSub: Subscription;
    _routeParamsSub: Subscription;

    constructor(private _broadcastStore: BroadcastStore, private _route: ActivatedRoute,
                private _cd: ChangeDetectorRef) {
        this._routeParamsSub = this._route.params.subscribe((params) => {
            let id = parseInt(params['id'], 10);

            this._broadcastStore.markAsRead(id)
                .subscribe();

            this.loading = true;
            this._broadcastSub = this._broadcastStore.getBroadcasts(id)
                .subscribe((bcs) => {
                    this.broadcast = bcs.get(id);
                    if (this.broadcast) {
                        this.loading = false;
                    }
                    this._cd.markForCheck();
                });
        });

    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT
}
