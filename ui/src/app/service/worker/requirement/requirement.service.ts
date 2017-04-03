import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';


/**
 * Service to get requirements constants
 * Only used by RequirementStore
 */
@Injectable()
export class RequirementService {

    constructor(private _http: Http) {
    }

    /**
     * Get the list of available requirements
     * @returns {Observable<string[]>}
     */
    getRequirementsTypes(): Observable<string[]> {
        return this._http.get('/worker/model/capability/type').map(res => res.json());
    }
}
