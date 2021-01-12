import {Injectable} from '@angular/core';
import {List} from 'immutable';
import {BehaviorSubject, Observable} from 'rxjs';
import {RequirementService} from './requirement.service';

@Injectable()
export class RequirementStore {

    // List of all available requirements
    private _requirementsType: BehaviorSubject<List<string>> = new BehaviorSubject(List([]));

    constructor(private _requirementService: RequirementService) {
    }

    /**
     * /**
     * Get the list of all available requirement.
     *
     * @returns
     */
    getAvailableRequirements(): Observable<List<string>> {
        let store = this._requirementsType.getValue();
        // If the store is empty, fill it
        if (store.size === 0) {
            this._requirementService.getRequirementsTypes().subscribe( res => {
                this._requirementsType.next(store.push(...res));
            });
        }
        return new Observable<List<string>>(fn => this._requirementsType.subscribe(fn));
    }

     /**
      * Get the list of available requirements values for a type
      *
      * @param type Type of requirement
      * @returns
      */
    getRequirementsTypeValues(type: string): Observable<string[]> {
        return new Observable<string[]>(fn => this._requirementService.getRequirementsTypeValues(type).subscribe(fn));
    }

}
