import {Injectable} from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {Broadcast} from '../../model/broadcast.model';
import {Observable} from 'rxjs/Observable';
import {map} from 'rxjs/operators';

/**
 * Service to access Broadcast from API.
 */
@Injectable()
export class BroadcastService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Create an broadcast
     * @returns {Observable<Broadcast>}
     */
    createBroadcast(broadcast: Broadcast): Observable<Broadcast> {
        return this._http.post<Broadcast>('/broadcast', broadcast);
    }

    /**
     * Delete an broadcast
     * @returns {Observable<Broadcast>}
     */
    deleteBroadcast(broadcast: Broadcast): Observable<boolean> {
        return this._http.delete('/broadcast/' + broadcast.id).pipe(map(() => {
            return true;
        }));
    }

    /**
     * Update an broadcast
     * @returns {Observable<Broadcast>}
     */
    updateBroadcast(broadcast: Broadcast): Observable<Broadcast> {
        return this._http.put<Broadcast>('/broadcast/' + broadcast.id, broadcast);
    }

    /**
     * Get Broadcast by id
     * @returns {Observable<Broadcast>}
     */
    getBroadcastById(broadcastId: number): Observable<Broadcast> {
        return this._http.get<Broadcast>('/broadcast/' + broadcastId);
    }


    /**
     * Update a broadcast to mark as read for a user
     * @returns {Observable<null>}
     */
    markAsRead(broadcastId: number): Observable<boolean> {
        return this._http.post<null>('/broadcast/' + broadcastId + '/mark', {}).pipe(map(() => true));
    }

    /**
     * Get the list of availablen broadcasts
     * @returns {Observable<Broadcast[]>}
     */
    getBroadcasts(): Observable<Array<Broadcast>> {
        return this._http.get<Array<Broadcast>>('/broadcast');
    }
}
