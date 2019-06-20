
import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { AuthConsumerSigninResponse } from 'app/model/authentication.model';
import { Observable } from 'rxjs';

@Injectable()
export class ApplicationService {

  constructor(
    private _http: HttpClient
  ) { }

  signin(consumerType: string, code: string, state: string): Observable<AuthConsumerSigninResponse> {
    return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/${consumerType}/signin`, {});
  }
}
