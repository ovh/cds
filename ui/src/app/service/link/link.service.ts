import {Observable} from "rxjs";
import {AuthDriverSigningRedirect} from "../../model/authentication.model";
import {Injectable} from "@angular/core";
import {HttpClient, HttpParams} from "@angular/common/http";

/**
 * Service to get downloads
 */
@Injectable()
export class LinkService {

    constructor(private _http: HttpClient) {
    }

    getDrivers(): Observable<Array<string>> {
        return this._http.get<Array<string>>('/link/driver');
    }

    askLink(consumerType: string, redirectURI: string): Observable<AuthDriverSigningRedirect> {
        let params = new HttpParams();
        if (redirectURI) {
            params = params.append('redirect_uri', redirectURI);
        }
        return this._http.post<AuthDriverSigningRedirect>(`/link/${consumerType}/ask`, {params})
    }

    link(consumerType: string, code: string, state: string): Observable<any> {
        return this._http.post(`/link/${consumerType}`, {
            code,
            state
        });
    }
}

