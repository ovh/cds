import { HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

@Injectable()
export class ProxyInterceptor implements HttpInterceptor {

    constructor() { }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        // Assets and version calls should not be redirect to CDS api
        if (req.url.indexOf('assets/i18n') !== -1 || req.url.indexOf('mon/version') !== -1) {
            return next.handle(req);
        }

        // If the request was send for cdn we will not add cdsapi path
        if (req.url.indexOf('cdscdn') !== -1) {
            return next.handle(req);
        }

        return next.handle(req.clone({
            url: './cdsapi' + req.url
        }));
    }
}
