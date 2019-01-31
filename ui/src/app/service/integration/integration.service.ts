import {HttpClient} from '@angular/common/http';
import { Injectable } from '@angular/core';
import {Observable} from 'rxjs';
import {IntegrationModel} from '../../model/integration.model';

@Injectable()
export class IntegrationService {

    constructor(private _http: HttpClient) { }

    getIntegrationModels(): Observable<Array<IntegrationModel>> {
        return this._http.get<Array<IntegrationModel>>('/integration/models');
    }

}
