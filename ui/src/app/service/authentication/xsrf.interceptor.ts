import { HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

@Injectable()
export class XSRFInterceptor implements HttpInterceptor {

    constructor() { }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        // Assets and version calls shoudl not be redirect to CDS api
        if (req.url.indexOf('assets/i18n') !== -1 || req.url.indexOf('mon/version') !== -1) {
            return next.handle(req);
        }

        return next.handle(req.clone({
            setHeaders: this.addHeader(),
            url: './cdsapi' + req.url
        }));
    }

    addHeader(): any {
        let headers = {};
        const xsrfCookie = this.getCookie('xsrf_token');
        if (xsrfCookie) {
            headers['X-XSRF-TOKEN'] = xsrfCookie;
        }
        return headers
    }

    getCookie(name: string): string {
        const nameLenPlus = (name.length + 1);
        return document.cookie
            .split(';')
            .map(c => c.trim())
            .filter(cookie => {
                return cookie.substring(0, nameLenPlus) === `${name}=`;
            })
            .map(cookie => {
                return decodeURIComponent(cookie.substring(nameLenPlus));
            })[0] || null;
    }
}
