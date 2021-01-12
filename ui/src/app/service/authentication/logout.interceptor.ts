import { HttpErrorResponse, HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { Injectable, NgZone } from '@angular/core';
import { NavigationExtras, Router } from '@angular/router';
import { Observable, throwError as observableThrowError } from 'rxjs';
import { catchError } from 'rxjs/operators';

@Injectable()
export class LogoutInterceptor implements HttpInterceptor {

    constructor(
        private _router: Router,
        private _ngZone: NgZone
    ) { }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        return next.handle(req).pipe(
            catchError(e => {
                if (e instanceof HttpErrorResponse) {
                    if (req.url.indexOf('auth') === -1 && e.status === 401) {
                        let navigationExtras: NavigationExtras = {
                            queryParams: {}
                        };

                        if (this._router.routerState.snapshot.url
                            && this._router.routerState.snapshot.url.indexOf('auth') === -1) {
                            navigationExtras.queryParams = { redirect: this._router.routerState.snapshot.url };
                        }

                        this._ngZone.run(_ => {
                            this._router.navigate(['/auth/signin'], navigationExtras);
                        });
                    }
                    return observableThrowError(e);
                }
            }));
    }
}
