import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';

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
     *
     * @returns
     */
    getRequirementsTypes(): Observable<string[]> {
        return this._http.get<string[]>('/requirement/types');
    }

    /**
     * Get the list of available requirements values for a type
     *
     * @param type Type of requirement
     * @returns
     */
    getRequirementsTypeValues(type: string): Observable<string[]> {
        return this._http.get<string[]>('/requirement/types/' + type);
    }
}
