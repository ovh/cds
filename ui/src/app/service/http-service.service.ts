import {Injectable} from '@angular/core';
import {Http, RequestOptions, ConnectionBackend, RequestOptionsArgs, Response, Headers} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {environment} from '../../environments/environment';
import {AuthentificationStore} from './auth/authentification.store';
import {Router, NavigationExtras} from '@angular/router';
import {ToastService} from '../shared/toast/ToastService';

@Injectable()
export class HttpService extends Http {

    /**
     * Get URL TOo API
     * @param url Path to transform
     * @returns {string}
     */
    static getUrl(url: string): string {
        return environment.apiURL + url;
    }

    constructor(_backend: ConnectionBackend, _defaultOptions: RequestOptions, private _toast: ToastService,
                private _authStore: AuthentificationStore, private _router: Router) {
        super(_backend, _defaultOptions);
    }

    get(url: string, options?: RequestOptionsArgs): Observable<Response> {
        if (url.indexOf('assets/i18n') !== -1) {
            return this.intercept(super.request(url, this.getRequestOptionArgs(options)));
        } else if (this._authStore.isConnected() || (options && !options.withCredentials)) {
            return this.intercept(super.get(HttpService.getUrl(url), this.getRequestOptionArgs(options)));
        } else {
            return Observable.throw({});
        }
    }

    post(url: string, body: any, options?: RequestOptionsArgs): Observable<Response> {
        return this.intercept(super.post(HttpService.getUrl(url), body, this.getRequestOptionArgs(options)));
    }

    put(url: string, body: any, options?: RequestOptionsArgs): Observable<Response> {
        return this.intercept(super.put(HttpService.getUrl(url), body, this.getRequestOptionArgs(options)));
    }

    delete(url: string, options?: RequestOptionsArgs): Observable<Response> {
        return this.intercept(super.delete(HttpService.getUrl(url), this.getRequestOptionArgs(options)));
    }

    intercept(observable: Observable<Response>): Observable<Response> {
        return observable.catch((err) => {
            if (err.status === 0) {
                this._toast.error('API Unreachable', '');
            } else {
                this._toast.error(err.statusText, JSON.parse(err._body).message);
            }
            if (err.status === 401) {
                this._authStore.removeUser();
                this._router.navigate(['/account/login']);
                return Observable.throw(err);
            } else {
                return Observable.throw(err);
            }
        });
    }

    getRequestOptionArgs(options?: RequestOptionsArgs): RequestOptionsArgs {
        if (options == null) {
            options = new RequestOptions();
        }
        if (options.headers == null) {
            options.headers = new Headers();
        }

        // ADD user AUTH
        let sessionToken = this._authStore.getSessionToken();
        if (sessionToken) {
            options.headers.append(this._authStore.localStorageSessionKey, sessionToken);
        } else {
            let user = this._authStore.getUser();
            if (user != null) {
                options.headers.append('Authorization', 'Basic ' + user.token);
            }
        }
        return options;
    }
}
