import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Download} from '../../model/download.model';
import {HttpClient} from '@angular/common/http';

/**
 * Service to get downloads
 */
@Injectable()
export class DownloadService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of available downloas
     * @returns {Observable<Download[]>}
     */
    getDownloads(): Observable<Array<Download>> {
        return this._http.get<Array<Download>>('/download');
    }
}
