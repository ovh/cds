import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { APIConfig } from 'app/model/config.service';

/**
 * Service to get config
 */
@Injectable()
export class ConfigService {
    constructor(private _http: HttpClient) {
    }

    getConfig(): Observable<any> {
        return this._http.get<any>('/config/user');
    }

    getAPIConfig() {
        return this._http.get<APIConfig>('/config/api');
    }
}
