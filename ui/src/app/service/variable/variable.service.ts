import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';

/**
 * Service to access Variable commons.
 * Only used by ProjectStore
 */
@Injectable()
export class VariableService {

    private variablesType: string[];

    constructor(private _http: Http) {
    }

    /**
     * Get variable type
     * @returns {any}
     */
    getTypesFromCache(): string[] {
        return this.variablesType;
    }

    /**
     * Get all types of variables
     * @returns {Observable<string[]>}
     */
    getTypesFromAPI(): Observable<string[]> {
        return this._http.get('/variable/type').map(res => {
            this.variablesType = res.json();
            return this.variablesType;
        });
    }

    /**
     * Get all available variable
     * @param key
     * @returns {Observable<Array<string>>}
     */
    getContextVariable(key: string): Observable<Array<string>> {
        return this._http.get('/suggest/variable/' + key).map(res => res.json());
    }
}
