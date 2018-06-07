import { Injectable } from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {PlatformModel} from '../../model/platform.model';
import {Observable} from 'rxjs';

@Injectable()
export class PlatformService {

    constructor(private _http: HttpClient) { }

    getPlatformModels(): Observable<Array<PlatformModel>> {
        return this._http.get<Array<PlatformModel>>('/platform/models');
    }

}
