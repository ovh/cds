import {Injectable} from "@angular/core";
import {HttpClient, HttpParams} from "@angular/common/http";
import {Observable} from "rxjs";
import {EntityFullName} from "../../model/entity.model";


@Injectable()
export class EntityService {

    constructor(
        private _http: HttpClient
    ) {
    }

    getEntities(entityType: string): Observable<Array<EntityFullName>> {
        return this._http.get<Array<EntityFullName>>(`/v2/entity/${entityType}`);
    }

}
