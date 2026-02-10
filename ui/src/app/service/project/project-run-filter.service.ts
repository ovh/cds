import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ProjectRunFilter } from 'app/model/project-run-filter.model';

@Injectable()
export class ProjectRunFilterService {
    constructor(private _http: HttpClient) {}

    /**
     * Liste tous les filtres partagés d'un projet
     */
    list(projectKey: string): Observable<Array<ProjectRunFilter>> {
        return this._http.get<Array<ProjectRunFilter>>(
            `/v2/project/${projectKey}/run-filter`
        );
    }

    /**
     * Crée un nouveau filtre partagé
     */
    create(projectKey: string, filter: ProjectRunFilter): Observable<ProjectRunFilter> {
        return this._http.post<ProjectRunFilter>(
            `/v2/project/${projectKey}/run-filter`,
            filter
        );
    }

    /**
     * Modifie un filtre partagé (actuellement : uniquement le champ order)
     */
    update(projectKey: string, filterName: string, filter: Partial<ProjectRunFilter>): Observable<ProjectRunFilter> {
        return this._http.put<ProjectRunFilter>(
            `/v2/project/${projectKey}/run-filter/${encodeURIComponent(filterName)}`,
            filter
        );
    }

    /**
     * Supprime un filtre partagé
     */
    delete(projectKey: string, filterName: string): Observable<void> {
        return this._http.delete<void>(
            `/v2/project/${projectKey}/run-filter/${encodeURIComponent(filterName)}`
        );
    }
}
