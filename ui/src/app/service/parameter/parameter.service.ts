import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {HttpClient} from '@angular/common/http';

/**
 * Service to access Parameter commons.
 */
@Injectable()
export class ParameterService {

    private parametersType: string[];

    constructor(private _http: HttpClient) {
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
        return this._http.get('/parameter/type');
    }
}
