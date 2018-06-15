
import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';
import {ModelPattern, WorkerModel} from '../../model/worker-model.model';

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
        return this._http.delete('/worker/model/' + workerModel.id).pipe(map(() => {
            return true;
        }));
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
    getWorkerModels(binary?: string): Observable<Array<WorkerModel>> {
        let params = new HttpParams();
        if (binary) {
          params = params.append('binary', binary);
        }

        return this._http.get<Array<WorkerModel>>('/worker/model', {params});
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
     * delete a worker model pattern
     * @returns {Observable<null>}
     */
    deleteWorkerModelPattern(type: string, name: string): Observable<null> {
        return this._http.delete<null>(`/worker/model/pattern/${type}/${name}`);
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
