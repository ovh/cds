import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Action, Usage } from 'app/model/action.model';
import { AuditAction } from 'app/model/audit.model';
import { Observable } from 'rxjs';

@Injectable()
export class ActionService {
    constructor(private _http: HttpClient) { }

    getAll(): Observable<Action[]> {
        return this._http.get<Action[]>('/action');
    }

    getAllBuiltin(): Observable<Action[]> {
        return this._http.get<Action[]>('/actionBuiltin');
    }

    getAllForProject(projectKey: string): Observable<Action[]> {
        return this._http.get<Action[]>(`/project/${projectKey}/action`);
    }

    getAllForGroup(groupName: string): Observable<Action[]> {
        return this._http.get<Action[]>(`/group/${groupName}/action`);
    }

    get(groupName: string, name: string): Observable<Action> {
        return this._http.get<Action>(`/action/${groupName}/${name}`);
    }

    getBuiltin(name: string): Observable<Action> {
        return this._http.get<Action>(`/actionBuiltin/${name}`);
    }

    getUsage(groupName: string, name: string): Observable<Usage> {
        return this._http.get<Usage>(`/action/${groupName}/${name}/usage`);
    }

    getBuiltinUsage(name: string): Observable<Usage> {
        return this._http.get<Usage>(`/actionBuiltin/${name}/usage`);
    }

    getAudits(groupName: string, name: string): Observable<Array<AuditAction>> {
        return this._http.get<Array<AuditAction>>(`/action/${groupName}/${name}/audit`);
    }

    rollbackAudit(groupName: string, actionName: string, auditID: number): Observable<Action> {
        return this._http.post<Action>(`/action/${groupName}/${actionName}/audit/${auditID}/rollback`, null);
    }

    add(action: Action): Observable<Action> {
        return this._http.post<Action>('/action', action);
    }

    update(old: Action, a: Action): Observable<Action> {
        return this._http.put<Action>(`/action/${old.group.name}/${old.name}`, a);
    }

    delete(groupName: string, name: string): Observable<Response> {
        return this._http.delete<Response>(`/action/${groupName}/${name}`);
    }
}
