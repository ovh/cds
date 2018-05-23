import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Broadcast} from '../../model/broadcast.model';
import * as immutable from 'immutable';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {BroadcastService} from './broadcast.service';

/**
 * Service to get broadcast
 */
@Injectable()
export class BroadcastStore {

    // List of all broadcasts.
    private _broadcasts: BehaviorSubject<immutable.Map<number, Broadcast>> = new BehaviorSubject(immutable.Map<number, Broadcast>());

    constructor(private _broadcastService: BroadcastService) {
    }

    create(broadcast: Broadcast): Observable<Broadcast> {
        return this._broadcastService.createBroadcast(broadcast).map(bc => {
            this.addBroadcastInCache(bc);
            return bc;
        });
    }

    addBroadcastInCache(bc: Broadcast): void {
        this._broadcasts.next(this._broadcasts.getValue().set(bc.id, bc));
    }

    delete(broadcast: Broadcast): Observable<boolean> {
        return this._broadcastService.deleteBroadcast(broadcast).map(b => {
           this.removeBroadcastFromCache(broadcast.id);
           return b;
        });
    }

    removeBroadcastFromCache(bcID: number): void {
        this._broadcasts.next(this._broadcasts.getValue().delete(bcID));
    }

    update(broadcast: Broadcast): Observable<Broadcast> {
        return this._broadcastService.updateBroadcast(broadcast).map(bc => {
            this.addBroadcastInCache(bc);
            return bc;
        });
    }

    markAsRead(broadcastId: number): Observable<boolean> {
        return this._broadcastService.markAsRead(broadcastId).map(b => {
            let bc = this._broadcasts.getValue().get(broadcastId);
            if (bc) {
                bc.read = true;
            }
            this.addBroadcastInCache(bc);
            return b;
        })
    }


    /**
     * Get the list of availablen broadcasts
     * @returns {Observable<Broadcast[]>}
     */
    getBroadcasts(id?: number): Observable<immutable.Map<number, Broadcast>> {
        if (id && !this._broadcasts.getValue().get(id)) {
            this._broadcastService.getBroadcastById(id).subscribe(b => {
                this.addBroadcastInCache(b);
            });
        } else if (this._broadcasts.getValue().size === 0) {
            this._broadcastService.getBroadcasts().subscribe(bcs => {
                let m = immutable.Map<number, Broadcast>();
                if (bcs) {
                    bcs.forEach(bc => {
                       m = m.set(bc.id, bc);
                    });
                    this._broadcasts.next(m);
                }
            });
        }
        return new Observable<immutable.Map<number, Broadcast>>(fn => this._broadcasts.subscribe(fn));
    }
}
