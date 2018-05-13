import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Broadcast} from '../../model/broadcast.model';
import {HttpClient} from '@angular/common/http';

/**
 * Service to get broadcast
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
        return this._http.delete('/broadcast/' + broadcast.id).map(() => {
            return true;
        });
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
    getBroadcastById(broadcastId: string): Observable<Broadcast> {
        return this._http.get<Broadcast>('/broadcast/' + broadcastId);
    }

    /**
     * Get the list of availablen broadcasts
     * @returns {Observable<Broadcast[]>}
     */
    getBroadcasts(): Observable<Array<Broadcast>> {
        return this._http.get<Array<Broadcast>>('/broadcast');
    }
}
