import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {DownloadableResource} from '../../model/download.model';

/**
 * Service to get downloads
 */
@Injectable()
export class DownloadService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of available downloas
     * @returns {Observable<DownloadableResource[]>}
     */
    getDownloads(): Observable<Array<DownloadableResource>> {
        return this._http.get<Array<DownloadableResource>>('/download');
    }
}
