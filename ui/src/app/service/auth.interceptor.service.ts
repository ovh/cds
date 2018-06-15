import {HttpEvent, HttpHandler, HttpInterceptor, HttpRequest} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {environment} from '../../environments/environment';
import {AuthentificationStore} from './auth/authentification.store';

@Injectable()
export class AuthentificationInterceptor implements  HttpInterceptor {

    constructor(private _authStore: AuthentificationStore) {}

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        if (req.url.indexOf('assets/i18n') !== -1) {
            const clone = req.clone();
            return next.handle(clone);
        } else {
            const clone = req.clone( { setHeaders: this.addAuthHeader(), url: environment.apiURL + req.url });
            return next.handle(clone);
        }
    }

    addAuthHeader(): any {
        if (this._authStore.isConnected()) {
            let sessionToken = this._authStore.getSessionToken();
            if (sessionToken) {
                return { 'Session-Token': sessionToken };
            } else {
                let user = this._authStore.getUser();
                if (user != null) {
                    return { 'Authorization': 'Basic ' + user.token};
                }
            }
        }
        return {}
    }
}
