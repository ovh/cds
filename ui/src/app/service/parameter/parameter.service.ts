
import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';

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
     *
     * @returns
     */
    getTypesFromCache(): string[] {
        return this.parametersType;
    }

    /**
     * Get all types of parameters
     *
     * @returns
     */
    getTypesFromAPI(): Observable<string[]> {
        return this._http.get<string[]>('/parameter/type').pipe(map( pts => {
            this.parametersType = <string[]>pts;
            return pts;
        }));
    }
}
