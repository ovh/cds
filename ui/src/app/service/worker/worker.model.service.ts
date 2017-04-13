import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {WorkerModel} from '../../model/worker.model';

/**
 * Service to get worker model
 */
@Injectable()
export class WorkerModelService {

    constructor(private _http: Http) {
    }

    /**
     * Get the list of available requirements
     * @returns {Observable<string[]>}
     */
    getWorkerModel(): Observable<Array<WorkerModel>> {
        return this._http.get('/worker/model').map(res => res.json());
    }
}
