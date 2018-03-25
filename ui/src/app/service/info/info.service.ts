import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Info} from '../../model/info.model';
import {HttpClient} from '@angular/common/http';

/**
 * Service to get info
 */
@Injectable()
export class InfoService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Create an info
     * @returns {Observable<Info>}
     */
    createInfo(info: Info): Observable<Info> {
        return this._http.post<Info>('/info', info);
    }

    /**
     * Delete an info
     * @returns {Observable<Info>}
     */
    deleteInfo(info: Info): Observable<boolean> {
        return this._http.delete('/info/' + info.id).map(() => {
            return true;
        });
    }

    /**
     * Update an info
     * @returns {Observable<Info>}
     */
    updateInfo(info: Info): Observable<Info> {
        return this._http.put<Info>('/info/' + info.id, info);
    }

    /**
     * Get Info by id
     * @returns {Observable<Info>}
     */
    getInfoById(infoId: string): Observable<Info> {
        return this._http.get<Info>('/info/' + infoId);
    }

    /**
     * Get the list of availablen infos
     * @returns {Observable<Info[]>}
     */
    getInfos(): Observable<Array<Info>> {
        return this._http.get<Array<Info>>('/info');
    }
}
