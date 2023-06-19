import {Injectable} from "@angular/core";
import {HttpClient, HttpParams} from "@angular/common/http";
import {Observable} from "rxjs";
import {EntityCheckResponse, EntityFullName} from "../../model/entity.model";


@Injectable()
export class EntityService {

    constructor(
        private _http: HttpClient
    ) {
    }

    getEntities(entityType: string): Observable<Array<EntityFullName>> {
        return this._http.get<Array<EntityFullName>>(`/v2/entity/${entityType}`);
    }

    checkEntity(entityType: string, payload: any): Observable<EntityCheckResponse> {
        return this._http.post<EntityCheckResponse>(`/v2/entity/${entityType}/check`, payload, {headers: {"Content-Type": "application/x-yaml"}});
    }

}
