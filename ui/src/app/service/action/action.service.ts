import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { Action, Usage } from '../../model/action.model';
import { AuditAction } from '../../model/audit.model';

@Injectable()
export class ActionService {
    constructor(private _http: HttpClient) { }

    getAll(): Observable<Action[]> {
        return this._http.get<Action[]>('/action');
    }

    getAllForProject(projectKey: string): Observable<Action[]> {
        return this._http.get<Action[]>(`/project/${projectKey}/action`);
    }

    get(groupName: string, name: string): Observable<Action> {
        return this._http.get<Action>(`/action/${groupName}/${name}`);
    }

    getUsage(groupName: string, name: string): Observable<Usage> {
        return this._http.get<Usage>(`/action/${groupName}/${name}/usage`);
    }

    getAudits(groupName: string, name: string): Observable<Array<AuditAction>> {
        return this._http.get<Array<AuditAction>>(`/action/${groupName}/${name}/audit`);
    }

    add(action: Action): Observable<Action> {
        return this._http.post<Action>('/action', action);
    }

    updateAction(name: string, action: Action): Observable<Action> {
        return this._http.put<Action>('/action/' + name, action);
    }

    update(old: Action, a: Action): Observable<Action> {
        return this._http.put<Action>(`/action/${old.group.name}/${old.name}`, a);
    }

    deleteAction(name: string): Observable<Response> {
        return this._http.delete<Response>('/action/' + name);
    }
}
