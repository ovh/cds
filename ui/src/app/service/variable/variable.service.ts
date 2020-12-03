
import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Variable} from 'app/model/variable.model';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';

/**
 * Service to access Variable commons.
 * Only used by ProjectStore
 */
@Injectable()
export class VariableService {

    private variablesType: string[];

    constructor(private _http: HttpClient) {
    }

    get(key: string): Observable<Variable[]> {
      return this._http.get<Variable[]>(`/project/${key}/variable`);
    }

    /**
     * Get variable type
     *
     * @returns
     */
    getTypesFromCache(): string[] {
        return this.variablesType;
    }

    /**
     * Get all types of variables
     *
     * @returns
     */
    getTypesFromAPI(): Observable<string[]> {
        return this._http.get<string[]>('/variable/type').pipe(map(vts => {
            this.variablesType = <string[]>vts;
            return vts;
        }));
    }

    /**
     * Get all available variable
     *
     * @param key
     * @returns
     */
    getContextVariable(key: string, pipelineId?: number): Observable<Array<string>> {
        let params = new HttpParams();
        if (pipelineId != null) {
            params = params.append('pipId', pipelineId.toString());
        }

        return this._http.get<Array<string>>('/suggest/variable/' + key, {params});
    }
}
