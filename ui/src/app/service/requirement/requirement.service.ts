import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {HttpClient} from '@angular/common/http';

/**
 * Service to get requirements constants
 * Only used by RequirementStore
 */
@Injectable()
export class RequirementService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the list of available requirements
     * @returns {Observable<string[]>}
     */
    getRequirementsTypes(): Observable<string[]> {
        return this._http.get<string[]>('/requirement/types');
    }

    /**
     * Get the list of available requirements values for a type
     * @param type Type of requirement
     * @returns {Observable<string[]>}
     */
    getRequirementsTypeValues(type: string): Observable<string[]> {
        return this._http.get<string[]>('/requirement/types/' + type);
    }
}
