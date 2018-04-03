import {Injectable} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {HttpErrorResponse, HttpEvent, HttpHandler, HttpInterceptor, HttpRequest} from '@angular/common/http';
import {Observable} from 'rxjs/Observable';
import {ToastService} from '../shared/toast/ToastService';
import {AuthentificationStore} from './auth/authentification.store';
import {Router} from '@angular/router';
import 'rxjs/add/observable/throw';

@Injectable()
export class LogoutInterceptor implements HttpInterceptor {

    constructor(
        private _toast: ToastService,
        private _authStore: AuthentificationStore,
        private _router: Router,
        private _translate: TranslateService) {
    }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(req).catch(e => {
            if (e instanceof HttpErrorResponse) {
                if (e.status === 0) {
                    this._toast.error('API Unreachable', '');
                } else if (e.status === 401) {
                    this._authStore.removeUser();
                    this._router.navigate(['/account/login']);
                } else {
                    // error formatted from CDS API
                    if (e.error && e.error.message) {
                        this._toast.error(e.statusText, e.error.message);
                    } else {
                        try {
                            this._toast.error(e.statusText, JSON.parse(e.message));
                        } catch (e) {
                            this._toast.error(e.statusText, this._translate.instant('common_error'));
                        }
                    }
                }
                return Observable.throw(e);
            }
        });
    }
}
