import {HttpClient} from '@angular/common/http';
import { Injectable } from '@angular/core';
import {Observable} from 'rxjs';
import {PlatformModel} from '../../model/platform.model';

@Injectable()
export class PlatformService {

    constructor(private _http: HttpClient) { }

    getPlatformModels(): Observable<Array<PlatformModel>> {
        return this._http.get<Array<PlatformModel>>('/platform/models');
    }

}
