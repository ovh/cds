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

    signout(): Observable<any> {
        return this._http.post('/auth/consumer/signout', null);
    }

    localSignup(fullname: string, email: string, username: string, password: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>('/auth/consumer/local/signup', {
            fullname,
            email,
            username,
            password
        });
    }

    localSignin(username: string, password: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>('/auth/consumer/local/signin', {
            username,
            password
        });
    }

    localVerify(token: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/verify`, {
            token
        });
    }

    localAskReset(email: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/askReset`, {
            email,
        });
    }

    localReset(token: string, password: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/reset`, {
            token,
            password
        });
    }
}
