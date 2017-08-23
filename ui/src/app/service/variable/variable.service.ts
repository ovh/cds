import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {HttpClient, HttpParams} from '@angular/common/http';

/**
 * Service to access Variable commons.
 * Only used by ProjectStore
 */
@Injectable()
export class VariableService {

    private variablesType: string[];

    constructor(private _http: HttpClient) {
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
        return this._http.get('/variable/type');
    }

    /**
     * Get all available variable
     * @param key
     * @returns {Observable<Array<string>>}
     */
    getContextVariable(key: string, pipelineId?: number): Observable<Array<string>> {
        let params = new HttpParams();
        if (pipelineId) {
            params.append('pipId', pipelineId.toString());
        }
        return this._http.get('/suggest/variable/' + key, {params: params});
    }
}
