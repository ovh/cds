
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Concurrency, ProjectConcurrencyRuns } from 'app/model/project.concurrency.model';
import { Project } from 'app/model/project.model';
import { VariableSet, VariableSetItem } from 'app/model/variablesets.model';
import { Observable } from 'rxjs';

@Injectable()
export class V2ProjectService {

    constructor(
        private _http: HttpClient
    ) { }

    getAll(): Observable<Array<Project>> {
        return this._http.get<Array<Project>>('/v2/project');
    }

    get(key: string): Observable<Project> {
        return this._http.get<Project>(`/v2/project/${key}`);
    }

    put(p: Project): Observable<Project> {
        return this._http.put<Project>(`/v2/project/${p.key}`, p);
    }

    delete(key: string): Observable<any> {
        return this._http.delete(`/v2/project/${key}`);
    }

    getConcurrencyRuns(key: string, name: string): Observable<Array<ProjectConcurrencyRuns>> {
        return this._http.get<Array<ProjectConcurrencyRuns>>(`/v2/project/${key}/concurrency/${name}/runs`)
    }

    getConcurrencies(key: string): Observable<Array<Concurrency>> {
        return this._http.get<Array<Concurrency>>(`/v2/project/${key}/concurrency`)
    }

    createConcurrency(key: string, concurrency: Concurrency): Observable<Concurrency> {
        return this._http.post<Concurrency>(`/v2/project/${key}/concurrency`, concurrency);
    }

    updateConcurrency(key: string, concurrency: Concurrency): Observable<Concurrency> {
        return this._http.put<Concurrency>(`/v2/project/${key}/concurrency/${concurrency.name}`, concurrency);
    }

    deleteConcurrency(key: string, vsName: string): Observable<any> {
        let params = new HttpParams();
        params.set('force', 'true');
        return this._http.delete(`/v2/project/${key}/concurrency/${vsName}`, { params });
    }


    getVariableSets(key: string): Observable<Array<VariableSet>> {
        return this._http.get<Array<VariableSet>>(`/v2/project/${key}/variableset`)
    }

    getVariableSet(key: string, vsName: string): Observable<VariableSet> {
        return this._http.get<VariableSet>(`/v2/project/${key}/variableset/${vsName}`)
    }

    createVariableSet(key: string, set: VariableSet): Observable<VariableSet> {
        return this._http.post<VariableSet>(`/v2/project/${key}/variableset`, set);
    }

    deleteVariableSet(key: string, vsName: string): Observable<any> {
        let params = new HttpParams();
        params.set('force', 'true');
        return this._http.delete(`/v2/project/${key}/variableset/${vsName}`, { params });
    }

    postVariableSetItem(key: string, vsName: string, vsItem: VariableSetItem): Observable<VariableSetItem> {
        return this._http.post<VariableSetItem>(`/v2/project/${key}/variableset/${vsName}/item`, vsItem)
    }

    deleteVariableSetItem(key: string, vsName: string, itemName: string): Observable<any> {
        return this._http.delete(`/v2/project/${key}/variableset/${vsName}/item/${itemName}`);
    }

    getKeys(key: string): Observable<Array<Key>> {
        return this._http.get<Array<Key>>(`/v2/project/${key}/keys`);
    }

    postKey(projectKey: string, key: Key): Observable<Key> {
        return this._http.post<Key>(`/v2/project/${projectKey}/keys`, key);
    }

    deleteKey(projectKey: string, keyName: string): Observable<any> {
        return this._http.delete(`/v2/project/${projectKey}/keys/${keyName}`);
    }

    getIntegrations(key: string): Observable<Array<ProjectIntegration>> {
        return this._http.get<Array<ProjectIntegration>>(`/v2/project/${key}/integrations`);
    }

    postIntegration(key: string, p: ProjectIntegration): Observable<ProjectIntegration> {
        return this._http.post<ProjectIntegration>(`/v2/project/${key}/integrations`, p);
    }

    putIntegration(key: string, p: ProjectIntegration): Observable<ProjectIntegration> {
        return this._http.put<ProjectIntegration>(`/v2/project/${key}/integrations/${p.name}`, p);
    }

    deleteIntegration(key: string, integrationName: string): Observable<any> {
        return this._http.delete(`/v2/project/${key}/integrations/${integrationName}`);
    }
}
