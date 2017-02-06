import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';

/**
 * Service to access Parameter commons.
 */
@Injectable()
export class ParameterService {

    private parametersType: string[];

    constructor(private _http: Http) {
    }

    /**
     * Get variable type
     * @returns {any}
     */
    getTypesFromCache(): string[] {
        return this.parametersType;
    }

    /**
     * Get all types of parameters
     * @returns {Observable<string[]>}
     */
    getTypesFromAPI(): Observable<string[]> {
        return this._http.get('/parameter/type').map(res => {
            this.parametersType = res.json();
            return this.parametersType;
        });
    }
}
