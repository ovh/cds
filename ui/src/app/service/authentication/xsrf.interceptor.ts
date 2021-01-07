import { HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

@Injectable()
export class XSRFInterceptor implements HttpInterceptor {

    constructor() { }

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        if (req.url.indexOf('cdsapi') === -1) {
            return next.handle(req);
        }

        return next.handle(req.clone({
            setHeaders: this.addHeader(),
            url: req.url
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
            .filter(cookie => cookie.substring(0, nameLenPlus) === `${name}=`)
            .map(cookie => decodeURIComponent(cookie.substring(nameLenPlus)))[0] || null;
    }
}
