import { HttpErrorResponse, HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ToastService } from 'app/shared/toast/ToastService';
import { SignoutCurrentUser } from 'app/store/authentication.action';
import { Observable, throwError as observableThrowError } from 'rxjs';
import { catchError } from 'rxjs/operators';

@Injectable()
export class ErrorInterceptor implements HttpInterceptor {

    constructor(
        private _toast: ToastService,
        private _translate: TranslateService,
        private _store: Store,
        private _router: Router) {
    }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(req).pipe(
            catchError(e => {
                if (e instanceof HttpErrorResponse) {
                    if (e.status === 0) {
                        this._toast.error('API Unreachable', '');
                        return observableThrowError(e);
                    }

                    // ignore error on auth/me used for auth pages and on cdscdn
                    if (req.url.indexOf('auth/me') !== -1 || req.url.indexOf('cdscdn/item') !== -1) {
                        return observableThrowError(e);
                    }

                    // error formatted from CDS API
                    if (!e.error) {
                        return observableThrowError(e);
                    }

                    if (e.error.message) {
                        // 194 is the error for "MFA required. See https://github.com/ovh/cds/blob/master/sdk/error.go#L205"
                        if (e.error.id === 194 && confirm(`${e.error.message}.\nDo you want to login using MFA ?`)) {
                            let currentURL = this._router.url;
                            this._store.dispatch(new SignoutCurrentUser()).subscribe(() => {
                                this._router.navigate(['/auth/ask-signin/corporate-sso'], {
                                    queryParams: {
                                        redirect_uri: currentURL,
                                        require_mfa: true
                                    }
                                });
                            });
                            return observableThrowError(e);
                        }
                        this._toast.errorHTTP(e.status, e.error.message, e.error.from, e.error.request_id);
                        return observableThrowError(e);
                    }

                    if (Array.isArray(e.error)) {
                        try {
                            let messages = e.error as Array<string>;
                            this._toast.error(e.statusText, messages.join(', '));
                        } catch (e) {
                            this._toast.error(e.statusText, this._translate.instant('common_error'));
                        }
                        return observableThrowError(e);
                    }

                    this._toast.error(e.statusText, this._translate.instant('common_error'));
                    return observableThrowError(e);
                }
            }));
    }
}
