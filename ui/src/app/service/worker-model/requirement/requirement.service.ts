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
        return this._http.get<string[]>('/worker/model/capability/type');
    }
}
