import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { AuthConsumerSigninResponse, AuthDriverManifests, AuthDriverSigningRedirect, AuthScope } from 'app/model/authentication.model';
import { Observable } from 'rxjs';

@Injectable()
export class AuthenticationService {
    constructor(
        private _http: HttpClient
    ) { }

    askSignin(consumerType: string, redirectURI: string, requireMFA: boolean): Observable<AuthDriverSigningRedirect> {
        let params = new HttpParams();
        if (redirectURI) {
            params = params.append('redirect_uri', redirectURI);
        }
        if (requireMFA) {
            params = params.append('require_mfa', String(requireMFA));
        }
        return this._http.get<AuthDriverSigningRedirect>(`/auth/consumer/${consumerType}/askSignin`, { params: params });
    }

    getDrivers(): Observable<AuthDriverManifests> {
        return this._http.get<AuthDriverManifests>('/auth/driver');
    }

    getScopes(): Observable<Array<AuthScope>> {
        return this._http.get<Array<string>>('/auth/scope').map(ss => {
            return ss.map(s => new AuthScope(s));
        });
    }

    signin(consumerType: string, code: string, state: string, init_token: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/${consumerType}/signin`, {
            code,
            state,
            init_token
        });
    }

    signout(): Observable<any> {
        return this._http.post('/auth/consumer/signout', null);
    }

    localSignup(fullname: string, email: string, username: string, password: string, init_token: string):
        Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>('/auth/consumer/local/signup', {
            fullname,
            email,
            username,
            password,
            init_token
        });
    }

    localSignin(username: string, password: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>('/auth/consumer/local/signin', {
            username,
            password
        });
    }

    ldapSignin(bind: string, password: string, init_token: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>('/auth/consumer/ldap/signin', {
            bind,
            password,
            init_token
        });
    }

    localVerify(token: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/verify`, {
            token
        });
    }

    localAskReset(email?: string): Observable<AuthConsumerSigninResponse> {
        let req = email ? { email } : {};
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/askReset`, req);
    }

    localReset(token: string, password: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/reset`, {
            token,
            password
        });
    }
}
