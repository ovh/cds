import {Component} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {BroadcastStore} from '../../../service/broadcast/broadcast.store';
import {Broadcast} from 'app/model/broadcast.model';
import {Subscription} from 'rxjs';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-broadcast-details',
    templateUrl: './broadcast.details.component.html',
    styleUrls: ['./broadcast.details.component.scss']
})
@AutoUnsubscribe()
export class BroadcastDetailsComponent {

    broadcast: Broadcast;
    loading = true;

    _broadcastSub: Subscription;
    _routeParamsSub: Subscription;

    constructor(private _broadcastStore: BroadcastStore, private _route: ActivatedRoute) {
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
                });
        });

    }

}
