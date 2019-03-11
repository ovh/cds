import { registerLocaleData } from '@angular/common';
import localeEN from '@angular/common/locales/en';
import localeFR from '@angular/common/locales/fr';
import { Component, NgZone, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationEnd, ResolveEnd, ResolveStart, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Observable } from 'rxjs';
import { bufferTime, filter, map, mergeMap } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import * as format from 'string-format-obj';
import { environment } from '../environments/environment';
import { AppService } from './app.service';
import { Event } from './model/event.model';
import { AuthentificationStore } from './service/auth/authentification.store';
import { LanguageStore } from './service/language/language.store';
import { NotificationService } from './service/notification/notification.service';
import { ThemeStore } from './service/theme/theme.store';
import { AutoUnsubscribe } from './shared/decorator/autoUnsubscribe';
import { ToastService } from './shared/toast/ToastService';
import { CDSSharedWorker } from './shared/worker/shared.worker';
import { CDSWebWorker } from './shared/worker/web.worker';
import { CDSWorker } from './shared/worker/worker';

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
@AutoUnsubscribe()
export class AppComponent implements OnInit {
    open: boolean;
    isConnected = false;
    versionWorker: CDSWebWorker;
    sseWorker: CDSWorker;
    heartbeatToken: number;
    zone: NgZone;
    currentVersion = 0;
    showUIUpdatedBanner = false;
    languageSubscriber: Subscription;
    themeSubscriber: Subscription;
    versionWorkerSubscription: Subscription;
    _routerSubscription: Subscription;
    _routerNavEndSubscription: Subscription;
    _sseSubscription: Subscription;
    displayResolver = false;
    toasterConfig: any;
    lastPing: number;
    currentTheme: string;

    constructor(
        _translate: TranslateService,
        private _language: LanguageStore,
        private _theme: ThemeStore,
        private _activatedRoute: ActivatedRoute,
        private _titleService: Title,
        private _authStore: AuthentificationStore,
        private _router: Router,
        private _notification: NotificationService,
        private _appService: AppService,
        private _toastService: ToastService
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
        this.toasterConfig = this._toastService.getConfig();
        _translate.addLangs(['en', 'fr']);
        _translate.setDefaultLang('en');
        let browserLang = navigator.language.match(/fr/) ? 'fr' : 'en';
        _translate.use(browserLang.match(/en|fr/) ? browserLang : 'en');
        registerLocaleData(browserLang.match(/fr/) ? localeFR : localeEN);

        this.languageSubscriber = this._language.get().subscribe(l => {
            if (l) {
                _translate.use(l);
            } else {
                _language.set(browserLang.match(/en|fr/) ? browserLang : 'en');
            }
        });

        this.themeSubscriber = this._theme.get().subscribe(t => {
            if (t) {
                this.currentTheme = t;
            } else {
                _theme.set('light');
            }
        });

        this._notification.requestPermission();
    }

    ngOnInit(): void {
        this._authStore.getUserlst().subscribe(user => {
            if (!user) {
                this.isConnected = false;
                this.stopWorker(this.sseWorker, null);
            } else {
                this.isConnected = true;
                this.startSSE();
            }
            this.startVersionWorker();
        });

        this._routerSubscription = this._router.events
            .pipe(filter((event) => event instanceof ResolveStart || event instanceof ResolveEnd))
            .subscribe(e => {
                if (e instanceof ResolveStart) {
                    this.displayResolver = true;
                }
                if (e instanceof ResolveEnd) {
                    this.displayResolver = false;
                }
            });

        this._routerNavEndSubscription = this._router.events
            .pipe(filter((event) => event instanceof NavigationEnd))
            .pipe(map(() => this._activatedRoute))
            .pipe(map((route) => {
                let params = {};
                while (route.firstChild) {
                    route = route.firstChild;
                    Object.assign(params, route.snapshot.params, route.snapshot.queryParams);
                }
                this._appService.updateRoute(params);
                return { route, params: Observable.of(params) };
            }))
            .pipe(filter((event) => event.route.outlet === 'primary'))
            .pipe(mergeMap((event) => Observable.zip(event.route.data, event.params)))
            .subscribe((routeData) => {
                if (!Array.isArray(routeData) || routeData.length < 2) {
                    return;
                }
                if (routeData[0]['title']) {
                    let title = format(routeData[0]['title'], routeData[1]);
                    this._titleService.setTitle(title);
                } else {
                    this._titleService.setTitle('CDS');
                }
            });
    }

    stopWorker(w: CDSWorker, s: Subscription): void {
        if (w) {
            w.stop();
        }
        if (s) {
            s.unsubscribe();
        }
    }

    startSSE(): void {
        if (this.sseWorker) {
            this.stopWorker(this.sseWorker, null);
        }
        let authKey: string;
        let authValue: string;
        let user = this._authStore.getUser();
        // ADD user AUTH
        let sessionToken = this._authStore.getSessionToken();
        if (sessionToken) {
            authKey = this._authStore.localStorageSessionKey;
            authValue = sessionToken;
        } else if (user) {
            authKey = 'Authorization';
            authValue = 'Basic ' + user.token;
        } else {
            return;
        }

        if (window['SharedWorker']) {
            this.sseWorker = new CDSSharedWorker('./assets/worker/sharedWorker.js');
            if (this.heartbeatToken !== 0) {
                clearInterval(this.heartbeatToken);
            }

            this.heartbeatToken = window.setInterval(() => {
                let d = (new Date()).getTime();
                if (this.lastPing !== 0 && (d - this.lastPing) > 11000) {
                    // If no ping in the last 11s restart SSE
                    this.startSSE();
                }
            }, 2000);
        } else {
            this.sseWorker = new CDSWebWorker('./assets/worker/webWorker.js');
        }

        this.sseWorker.start({
            headAuthKey: authKey,
            headAuthValue: authValue,
            urlSubscribe: environment.apiURL + '/events/subscribe',
            urlUnsubscribe: environment.apiURL + 'events/unsubscribe',
            sseURL: environment.apiURL + '/events',
            pingURL: environment.apiURL + '/user/logged'
        });
        this._sseSubscription = this.sseWorker.response()
            .pipe(
                filter((e) => e != null),
                bufferTime(2000),
                filter((events) => events.length !== 0)
            )
            .subscribe((events) => {
                this.zone.run(() => {
                    for (let e of events) {
                        if (e.healthCheck != null) {
                            this.lastPing = (new Date()).getTime();
                            // 0 = CONNECTING, 1 = OPEN, 2 = CLOSED
                            if (e.healthCheck > 1) {
                                // Reopen SSE
                                this.startSSE();
                            }
                        } else {
                            this._appService.manageEvent(<Event>e);
                        }
                    }
                });
            });
    }


    startVersionWorker(): void {
        this.stopWorker(this.versionWorker, this.versionWorkerSubscription);
        this.versionWorker = new CDSWebWorker('./assets/worker/web/version.js');
        this.versionWorker.start({});
        this.versionWorker.response().subscribe(msg => {
            if (msg) {
                this.zone.run(() => {
                    let versionJSON = Number(JSON.parse(msg).version);
                    if (this.currentVersion === 0) {
                        this.currentVersion = versionJSON;
                    }
                    if (this.currentVersion < versionJSON) {
                        this.showUIUpdatedBanner = true;
                    }
                });
            }
        });
    }

    refresh(): void {
        this.zone.runOutsideAngular(() => {
            location.reload(true);
        });
    }
}
