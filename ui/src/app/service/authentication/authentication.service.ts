import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { AuthConsumerSigninResponse, AuthDriverManifest } from 'app/model/authentication.model';
import { Observable } from 'rxjs';

@Injectable()
export class AuthenticationService {
  constructor(
    private _http: HttpClient
  ) { }

  getDrivers(): Observable<Array<AuthDriverManifest>> {
    return this._http.get<Array<AuthDriverManifest>>('/auth/driver');
  }

  signin(consumerType: string, code: string, state: string): Observable<AuthConsumerSigninResponse> {
    return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/${consumerType}/signin`, {
      code,
      state
    });
  }

  localSignin(consumerType: string, username: string, password: string): Observable<AuthConsumerSigninResponse> {
    return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/${consumerType}/signin`, {
      username,
      password
    });
  }

  localVerify(token: string): Observable<AuthConsumerSigninResponse> {
    return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/verify`, {
      token
    });
  }
}
