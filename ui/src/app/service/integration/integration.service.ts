import {HttpClient} from '@angular/common/http';
import { Injectable } from '@angular/core';
import {IntegrationModel} from 'app/model/integration.model';
import {Observable} from 'rxjs';

@Injectable()
export class IntegrationService {

    constructor(private _http: HttpClient) { }

    getIntegrationModels(): Observable<Array<IntegrationModel>> {
        return this._http.get<Array<IntegrationModel>>('/integration/models');
    }

}
