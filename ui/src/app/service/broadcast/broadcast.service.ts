import {Injectable} from '@angular/core';
import {Observable, BehaviorSubject} from 'rxjs';
import {map, share, flatMap} from 'rxjs/operators';
import {Broadcast} from '../../model/broadcast.model';
import {HttpClient} from '@angular/common/http';

/**
 * Service to get broadcast
 */
@Injectable()
export class BroadcastService {

    // List of all broadcasts.
    private _broadcasts: BehaviorSubject<Array<Broadcast>> = new BehaviorSubject(Array<Broadcast>());

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
        return this._http.get<Broadcast>('/broadcast/' + broadcastId)
            .pipe(
                map((broadcast) => {
                    this._broadcasts.next(this.resync(broadcast));
                    return broadcast;
                })
            );
    }

    /**
     * Update a broadcast to mark as read for a user
     * @returns {Observable<null>}
     */
    markAsRead(broadcastId: number): Observable<null> {
        return this._http.post<null>('/broadcast/' + broadcastId + '/mark', {})
          .pipe(
            map(() => {
                let broadcasts = this._broadcasts.getValue();
                if (Array.isArray(broadcasts) && broadcasts.length) {
                    broadcasts = broadcasts.map((br) => {
                        if (br.id === broadcastId) {
                            br.read = true;
                        }
                        return br;
                    });
                }
                this._broadcasts.next(broadcasts);

                return null;
            })
          );
    }

    /**
     * Get the list of availablen broadcasts
     * @returns {Observable<Broadcast[]>}
     */
    getBroadcasts(): Observable<Array<Broadcast>> {
        return this._http.get<Array<Broadcast>>('/broadcast')
            .pipe(
                share(),
                map((broadcasts) => {
                    this._broadcasts.next(broadcasts);
                    return broadcasts;
                })
            );
    }

    /**
     * Get the list of availablen broadcasts
     * @returns {Observable<Broadcast[]>}
     */
    getBroadcastsListener(): Observable<Array<Broadcast>> {
        let broadcasts = this._broadcasts.getValue();
        if (!Array.isArray(broadcasts) || !broadcasts.length) {
            return this.getBroadcasts()
                .pipe(flatMap(() => new Observable<Array<Broadcast>>(fn => this._broadcasts.subscribe(fn))));
        }
        return new Observable<Array<Broadcast>>(fn => this._broadcasts.subscribe(fn));
    }

    private resync(broadcast: Broadcast): Array<Broadcast> {
      let broadcasts = this._broadcasts.getValue();
      if (Array.isArray(broadcasts) && broadcasts.length) {
          broadcasts = broadcasts.map((br) => {
              if (br.id === broadcast.id) {
                  return broadcast;
              }
              return br;
          });
      }
      return broadcasts;
    }
}
