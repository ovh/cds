import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {WorkerModel} from '../../model/worker-model.model';
import {HttpClient} from '@angular/common/http';

/**
 * Service to get worker model
 */
@Injectable()
export class WorkerModelService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Create a worker model
     * @returns {Observable<WorkerModel>}
     */
    createWorkerModel(workerModel: WorkerModel): Observable<WorkerModel> {
        return this._http.post('/worker/model', workerModel);
    }

    /**
     * Delete a worker model
     * @returns {Observable<WorkerModel>}
     */
    deleteWorkerModel(workerModel: WorkerModel): Observable<boolean> {
        return this._http.delete('/worker/model/' + workerModel.id).map(() => {
            return true;
        });
    }

    /**
     * Update a worker model
     * @returns {Observable<WorkerModel>}
     */
    updateWorkerModel(workerModel: WorkerModel): Observable<WorkerModel> {
        return this._http.put('/worker/model/' + workerModel.id, workerModel);
    }

    /**
     * Get the list of available requirements
     * @returns {Observable<WorkerModel>}
     */
    getWorkerModelByName(workerModelName: string): Observable<WorkerModel> {
        return this._http.get('/worker/model?name=' + workerModelName);
    }

    /**
     * Get the list of available worker models
     * @returns {Observable<WorkerModel[]>}
     */
    getWorkerModels(): Observable<Array<WorkerModel>> {
        return this._http.get('/worker/model');
    }

    /**
     * Get the list of available worker model type
     * @returns {Observable<string[]>}
     */
    getWorkerModelTypes(): Observable<Array<string>> {
        return this._http.get('/worker/model/type');
    }

    /**
     * Get the list of available worker model communication
     * @returns {Observable<string[]>}
     */
    getWorkerModelCommunications(): Observable<Array<string>> {
        return this._http.get('/worker/model/communication');
    }
}
