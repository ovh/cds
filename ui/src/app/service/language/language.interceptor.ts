import { HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { LanguageStore } from './../language/language.store';

@Injectable()
export class LanguageInterceptor implements HttpInterceptor {
  languageHeader: string;

  constructor(
    private _language: LanguageStore
  ) {
    this.languageHeader = 'en-US';

    this._language.get().subscribe(l => {
      if (l) {
        this.languageHeader = l;
      }
    });
  }

  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    if (req.url.indexOf('assets/i18n') !== -1) {
      return next.handle(req.clone());
    }

    return next.handle(req.clone({
      setHeaders: { 'Accept-Language': this.languageHeader }
    }));
  }
}
