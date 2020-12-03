import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Broadcast } from 'app/model/broadcast.model';
import { Observable } from 'rxjs/Observable';
import { map } from 'rxjs/operators';

@Injectable()
export class BroadcastService {
    private broadcasts = [
        { name: 'info', value: 'info' },
        { name: 'warning', value: 'warning' }
    ];

    constructor(private _http: HttpClient) { }

    getBroadcastLevels() {
        return this.broadcasts;
    }

    createBroadcast(broadcast: Broadcast): Observable<Broadcast> {
        return this._http.post<Broadcast>('/broadcast', broadcast);
    }

    deleteBroadcast(broadcast: Broadcast): Observable<boolean> {
        return this._http.delete(`/broadcast/${broadcast.id}`).pipe(map(() => true));
    }

    updateBroadcast(broadcast: Broadcast): Observable<Broadcast> {
        return this._http.put<Broadcast>(`/broadcast/${broadcast.id}`, broadcast);
    }

    getBroadcastById(broadcastId: number): Observable<Broadcast> {
        return this._http.get<Broadcast>(`/broadcast/${broadcastId}`);
    }

    markAsRead(broadcastId: number): Observable<boolean> {
        return this._http.post<null>(`/broadcast/${broadcastId}/mark`, {}).pipe(map(() => true));
    }

    getBroadcasts(): Observable<Array<Broadcast>> {
        return this._http.get<Array<Broadcast>>('/broadcast');
    }
}
