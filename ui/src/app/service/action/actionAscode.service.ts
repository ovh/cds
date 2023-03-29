import {HttpClient, HttpParams} from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Action, Usage } from 'app/model/action.model';
import { AuditAction } from 'app/model/audit.model';
import { Observable } from 'rxjs';
import {ActionAsCode} from "../../model/action.ascode.model";

@Injectable()
export class ActionAsCodeService {
    constructor(private _http: HttpClient) { }

    get(projectKey: string, vcsIdentifier: string, repositoryIdentifier: string, actionName: string, branch?: string): Observable<ActionAsCode> {
        let params = new HttpParams();
        if (branch) {
            params = params.append('branch', branch);
        }
        let encodedRepo = encodeURIComponent(repositoryIdentifier);
        return this._http.get<ActionAsCode>(`/v2/project/${projectKey}/vcs/${vcsIdentifier}/repository/${encodedRepo}/action/${actionName}`, {params});
    }
}
