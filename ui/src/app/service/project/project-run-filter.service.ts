import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ProjectRunFilter } from 'app/model/project-run-filter.model';

@Injectable()
export class ProjectRunFilterService {
    constructor(private _http: HttpClient) {}

    /**
     * List all shared filters for a project
     */
    list(projectKey: string): Observable<Array<ProjectRunFilter>> {
        return this._http.get<Array<ProjectRunFilter>>(
            `/v2/project/${projectKey}/run-filter`
        );
    }

    /**
     * Create a new shared filter
     */
    create(projectKey: string, filter: ProjectRunFilter): Observable<ProjectRunFilter> {
        return this._http.post<ProjectRunFilter>(
            `/v2/project/${projectKey}/run-filter`,
            filter
        );
    }

    /**
     * Update a shared filter (currently: only the order field)
     */
    update(projectKey: string, filterName: string, filter: Partial<ProjectRunFilter>): Observable<ProjectRunFilter> {
        return this._http.put<ProjectRunFilter>(
            `/v2/project/${projectKey}/run-filter/${encodeURIComponent(filterName)}`,
            filter
        );
    }

    /**
     * Delete a shared filter
     */
    delete(projectKey: string, filterName: string): Observable<void> {
        return this._http.delete<void>(
            `/v2/project/${projectKey}/run-filter/${encodeURIComponent(filterName)}`
        );
    }
}
