import {Injectable} from '@angular/core';
import {HttpErrorResponse, HttpEvent, HttpHandler, HttpInterceptor, HttpRequest} from '@angular/common/http';
import {Observable} from 'rxjs/Observable';
import {ToastService} from '../shared/toast/ToastService';
import {AuthentificationStore} from './auth/authentification.store';
import {Router} from '@angular/router';

@Injectable()
export class LogoutInterceptor implements HttpInterceptor {

    constructor(private _toast: ToastService, private _authStore: AuthentificationStore, private _router: Router) {
    }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(req).catch(e => {
            if (e instanceof HttpErrorResponse) {
                if (e.status === 0) {
                    this._toast.error('API Unreachable', '');
                } else if (e.status === 401) {
                    this._authStore.removeUser();
                    this._router.navigate(['/account/login']);
                    return Observable.throw(e);
                } else {
                    // error formatted from CDS API
                    if (e.error && e.error.message) {
                        this._toast.error(e.statusText, e.error.message);
                    } else {
                        this._toast.error(e.statusText, JSON.parse(e.message));
                    }
                }
            }
        });
    }
}
