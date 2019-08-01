import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Config} from 'app/model/config.model';
import {Observable} from 'rxjs';

/**
 * Service to get config
 */
@Injectable()
export class ConfigService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the config (url api / url ui)
     * @returns {Observable<Config>}
     */
    getConfig(): Observable<Config> {
        return this._http.get<Config>('/config/user');
    }
}
