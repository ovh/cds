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
                    this._toast.error(e.statusText, JSON.parse(e.message));
                }
            }
        });
    }

    /*

     return observable.catch((err) => {
     if (err.status === 0) {
     this._toast.error('API Unreachable', '');
     } else {
     this._toast.error(err.statusText, JSON.parse(err._body).message);
     }
     if (err.status === 401) {
     let navigationExtras: NavigationExtras = {
     queryParams: {
     redirect: window.location.pathname + window.location.search
     }
     };

     this._authStore.removeUser();
     this._router.navigate(['/account/login'], navigationExtras);
     return Observable.throw(err);
     } else {
     return Observable.throw(err);
     }
     });


     */
}