import { HttpErrorResponse, HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { NavigationExtras, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { ToastService } from 'app/shared/toast/ToastService';
import { Observable, throwError as observableThrowError } from 'rxjs';
import { catchError } from 'rxjs/operators';

@Injectable()
export class LogoutInterceptor implements HttpInterceptor {

    constructor(
        private _toast: ToastService,
        private _router: Router,
        private _translate: TranslateService) {
    }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(req).pipe(
            catchError(e => {
                if (e instanceof HttpErrorResponse) {
                    if (e.status === 0) {
                        this._toast.error('API Unreachable', '');
                    } else if (req.url.indexOf('auth') === -1 && e.status === 401) {
                        let navigationExtras: NavigationExtras = {
                            queryParams: {}
                        };

                        if (this._router.routerState.snapshot.url
                            && this._router.routerState.snapshot.url.indexOf('auth') === -1) {
                            navigationExtras.queryParams = { redirect: this._router.routerState.snapshot.url };
                        }

                        this._router.navigate(['/auth/signin'], navigationExtras);
                    } else if (req.url.indexOf('auth/me') === -1) { // ignore error on auth/me used for auth pages
                        console.log(e)
                        // error formatted from CDS API
                        if (e.error) {
                            if (e.error.message) {
                                this._toast.error(e.statusText, e.error.message);
                            } else if (Array.isArray(e.error)) {
                                try {
                                    let messages = e.error as Array<string>;
                                    this._toast.error(e.statusText, messages.join(', '));
                                } catch (e) {
                                    this._toast.error(e.statusText, this._translate.instant('common_error'));
                                }
                            } else {
                                this._toast.error(e.statusText, this._translate.instant('common_error'));
                            }
                        }
                    }
                    return observableThrowError(e);
                }
            }));
    }
}
