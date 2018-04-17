import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {WorkerModel, ModelPattern} from '../../model/worker-model.model';
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
        return this._http.post<WorkerModel>('/worker/model', workerModel);
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
        return this._http.put<WorkerModel>('/worker/model/' + workerModel.id, workerModel);
    }

    /**
     * Get the list of available requirements
     * @returns {Observable<WorkerModel>}
     */
    getWorkerModelByName(workerModelName: string): Observable<WorkerModel> {
        return this._http.get<WorkerModel>('/worker/model?name=' + workerModelName);
    }

    /**
     * Get the list of available worker models
     * @returns {Observable<WorkerModel[]>}
     */
    getWorkerModels(): Observable<Array<WorkerModel>> {
        return this._http.get<Array<WorkerModel>>('/worker/model');
    }

    /**
     * Create a worker model pattern
     * @returns {Observable<ModelPattern>}
     */
    createWorkerModelPattern(workerModelPattern: ModelPattern): Observable<ModelPattern> {
        return this._http.post<ModelPattern>('/worker/model/pattern', workerModelPattern);
    }

    /**
     * update a worker model pattern
     * @returns {Observable<ModelPattern>}
     */
    updateWorkerModelPattern(type: string, name: string, workerModelPattern: ModelPattern): Observable<ModelPattern> {
        return this._http.put<ModelPattern>(`/worker/model/pattern/${type}/${name}`, workerModelPattern);
    }

    /**
     * Get the list of available worker model patterns
     * @returns {Observable<ModelPattern[]>}
     */
    getWorkerModelPatterns(): Observable<Array<ModelPattern>> {
        return this._http.get<Array<ModelPattern>>('/worker/model/pattern');
    }

    /**
     * Get worker model pattern
     * @returns {Observable<ModelPattern>}
     */
    getWorkerModelPattern(type: string, name: string): Observable<ModelPattern> {
        return this._http.get<ModelPattern>(`/worker/model/pattern/${type}/${name}`);
    }

    /**
     * Get the list of available worker model type
     * @returns {Observable<string[]>}
     */
    getWorkerModelTypes(): Observable<Array<string>> {
        return this._http.get<Array<string>>('/worker/model/type');
    }

    /**
     * Get the list of available worker model communication
     * @returns {Observable<string[]>}
     */
    getWorkerModelCommunications(): Observable<Array<string>> {
        return this._http.get<Array<string>>('/worker/model/communication');
    }
}
