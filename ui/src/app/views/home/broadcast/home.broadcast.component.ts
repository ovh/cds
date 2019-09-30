import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { Broadcast } from 'app/model/broadcast.model';
import { BroadcastStore } from 'app/service/services.module';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-home-broadcast',
    templateUrl: './home.broadcast.html',
    styleUrls: ['./home.broadcast.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class HomeBroadcastComponent {

    @Input() loading: boolean;
    @Input() broadcasts: Array<Broadcast>;


    constructor(private _cd: ChangeDetectorRef, private _broadcastStore: BroadcastStore) {

    }

    markAsRead(id: number) {
        for (let i = 0; i < this.broadcasts.length; i++) {
            if (this.broadcasts[i].id === id) {
                this.broadcasts[i].updating = true;
                break;
            }
        }
        this._broadcastStore.markAsRead(id)
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe();
    }
}
