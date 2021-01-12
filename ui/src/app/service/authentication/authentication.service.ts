import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import {
    AuthConsumerScopeDetail,
    AuthConsumerSigninResponse,
    AuthCurrentConsumerResponse,
    AuthDriverManifests,
    AuthDriverSigningRedirect
} from 'app/model/authentication.model';
import { Observable } from 'rxjs';

@Injectable()
export class AuthenticationService {
    constructor(
        private _http: HttpClient
    ) { }

    getMe(): Observable<AuthCurrentConsumerResponse> {
        return this._http.get<AuthCurrentConsumerResponse>('/auth/me');
    }

    askSignin(consumerType: string, origin: string, redirectURI: string, requireMFA: boolean): Observable<AuthDriverSigningRedirect> {
        let params = new HttpParams();
        if (origin) {
            params = params.append('origin', origin)
        }
        if (redirectURI) {
            params = params.append('redirect_uri', redirectURI);
        }
        if (requireMFA) {
            params = params.append('require_mfa', String(requireMFA));
        }
        return this._http.get<AuthDriverSigningRedirect>(`/auth/consumer/${consumerType}/askSignin`, { params });
    }

    getDrivers(): Observable<AuthDriverManifests> {
        return this._http.get<AuthDriverManifests>('/auth/driver');
    }

    getScopes(): Observable<Array<AuthConsumerScopeDetail>> {
        return this._http.get<Array<AuthConsumerScopeDetail>>('/auth/scope');
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

    detach(consumerType: string): Observable<any> {
        return this._http.post(`/auth/consumer/${consumerType}/detach`, null);
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

    ldapSignin(bind: string, password: string, init_token?: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>('/auth/consumer/ldap/signin', {
            bind,
            password,
            init_token
        });
    }

    localVerify(token: string, init_token?: string): Observable<AuthConsumerSigninResponse> {
        return this._http.post<AuthConsumerSigninResponse>(`/auth/consumer/local/verify`, {
            token,
            init_token
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
