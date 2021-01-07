import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {DownloadableResource} from 'app/model/download.model';
import {Observable} from 'rxjs';

/**
 * Service to get downloads
 */
@Injectable()
export class DownloadService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of available downloas
     *
     * @returns
     */
    getDownloads(): Observable<Array<DownloadableResource>> {
        return this._http.get<Array<DownloadableResource>>('/download');
    }
}
