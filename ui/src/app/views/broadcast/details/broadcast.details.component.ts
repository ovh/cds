import {Component} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {BroadcastService} from '../../../service/broadcast/broadcast.service';
import {Broadcast} from 'app/model/broadcast.model';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/operators';
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

    constructor(private _broadcastService: BroadcastService, private _route: ActivatedRoute) {
        this._routeParamsSub = this._route.params.subscribe((params) => {
            let id = parseInt(params['id'], 10);

            this._broadcastService.markAsRead(id)
                .subscribe();

            this.loading = true;
            this._broadcastSub = this._broadcastService.getBroadcastById(id)
                .pipe(
                    finalize(() => this.loading = false)
                )
                .subscribe((broadcast) => this.broadcast = broadcast);
        });

    }

}
