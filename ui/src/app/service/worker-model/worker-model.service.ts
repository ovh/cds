
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Pipeline } from 'app/model/pipeline.model';
import { ModelPattern, WorkerModel } from 'app/model/worker-model.model';
import { Observable } from 'rxjs';

@Injectable()
export class WorkerModelService {
    constructor(private _http: HttpClient) { }

    add(wm: WorkerModel): Observable<WorkerModel> {
        return this._http.post<WorkerModel>('/worker/model', wm);
    }

    import(workerModelStr: string, force = false): Observable<WorkerModel> {
        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');

        let params = new HttpParams();
        params = params.append('format', 'yaml');
        if (force) {
            params = params.append('force', 'true');
        }

        return this._http.post<WorkerModel>('/worker/model/import', workerModelStr, { params, headers });
    }

    export(groupName: string, name: string): Observable<string> {
        let params = new HttpParams();
        params = params.append('format', 'yaml');
        return this._http.get<string>(`/worker/model/${groupName}/${name}/export`, { params, responseType: <any>'text' });
    }

    delete(groupName: string, name: string): Observable<Response> {
        return this._http.delete<Response>(`/worker/model/${groupName}/${name}`);
    }

    update(old: WorkerModel, wm: WorkerModel): Observable<WorkerModel> {
        return this._http.put<WorkerModel>(`/worker/model/${old.group.name}/${old.name}`, wm);
    }

    get(groupName: string, name: string): Observable<WorkerModel> {
        return this._http.get<WorkerModel>(`/worker/model/${groupName}/${name}`);
    }

    getAll(state: string, binary?: string): Observable<Array<WorkerModel>> {
        let params = new HttpParams();
        if (binary) {
            params = params.append('binary', binary);
        }
        if (state) {
            params = params.append('state', state);
        }

        return this._http.get<Array<WorkerModel>>('/worker/model', { params });
    }

    getAllForProject(projectKey: string): Observable<Array<WorkerModel>> {
        return this._http.get<Array<WorkerModel>>(`/project/${projectKey}/worker/model`);
    }

    getAllForGroup(groupName: string): Observable<Array<WorkerModel>> {
        return this._http.get<Array<WorkerModel>>(`/group/${groupName}/worker/model`);
    }

    createPattern(mp: ModelPattern): Observable<ModelPattern> {
        return this._http.post<ModelPattern>('/worker/model/pattern', mp);
    }

    updatePattern(type: string, name: string, mp: ModelPattern): Observable<ModelPattern> {
        return this._http.put<ModelPattern>(`/worker/model/pattern/${type}/${name}`, mp);
    }

    deletePattern(type: string, name: string): Observable<null> {
        return this._http.delete<null>(`/worker/model/pattern/${type}/${name}`);
    }

    getPatterns(): Observable<Array<ModelPattern>> {
        return this._http.get<Array<ModelPattern>>('/worker/model/pattern');
    }

    getPattern(type: string, name: string): Observable<ModelPattern> {
        return this._http.get<ModelPattern>(`/worker/model/pattern/${type}/${name}`);
    }

    getTypes(): Observable<Array<string>> {
        return this._http.get<Array<string>>('/worker/model/type');
    }

    getUsage(groupName: string, name: string): Observable<Array<Pipeline>> {
        return this._http.get<Array<Pipeline>>(`/worker/model/${groupName}/${name}/usage`);
    }
}
