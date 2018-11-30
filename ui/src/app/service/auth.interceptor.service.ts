import {HttpEvent, HttpHandler, HttpInterceptor, HttpRequest} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {environment} from '../../environments/environment';
import {AuthentificationStore} from './auth/authentification.store';
import {LanguageStore} from './language/language.store';

@Injectable()
export class AuthentificationInterceptor implements  HttpInterceptor {

    languageHeader = 'en-US';
    constructor(private _authStore: AuthentificationStore, _language: LanguageStore) {
        _language.get().subscribe( l => {
            if (l) {
                this.languageHeader = l;
            }
        });
    }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        if (req.url.indexOf('assets/i18n') !== -1) {
            const clone = req.clone();
            return next.handle(clone);
        } else {
            const clone = req.clone( { setHeaders: this.addHeader(), url: environment.apiURL + req.url });
            return next.handle(clone);
        }
    }

    addHeader(): any {
        let headers = {};
        if (this._authStore.isConnected()) {
            let sessionToken = this._authStore.getSessionToken();
            headers['Session-Token'] = sessionToken;
        }
        headers['Accept-Language'] = this.languageHeader;
        return headers
    }
}
