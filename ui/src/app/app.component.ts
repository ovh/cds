import { registerLocaleData } from '@angular/common';
import localeEN from '@angular/common/locales/en';
import localeFR from '@angular/common/locales/fr';
import { Component, ElementRef, NgZone, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationEnd, NavigationStart, ResolveEnd, ResolveStart, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { EventService } from 'app/event.service';
import { GetCDSStatus } from 'app/store/cds.action';
import { CDSState } from 'app/store/cds.state';
import { Observable } from 'rxjs';
import { WebSocketSubject } from 'rxjs/internal-compatibility';
import { filter, map, mergeMap } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import * as format from 'string-format-obj';
import { AppService } from './app.service';
import { AuthentifiedUser } from './model/user.model';
import { LanguageStore } from './service/language/language.store';
import { NotificationService } from './service/notification/notification.service';
import { HelpService, MonitoringService } from './service/services.module';
import { ThemeStore } from './service/theme/theme.store';
import { AutoUnsubscribe } from './shared/decorator/autoUnsubscribe';
import { ToastService } from './shared/toast/ToastService';
import { AuthenticationState } from './store/authentication.state';
import { AddHelp } from './store/help.action';

declare var PACMAN: any;

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
@AutoUnsubscribe()
export class AppComponent implements OnInit, OnDestroy {
    open: boolean;
    isAPIAvailable: boolean;
    isConnected: boolean;
    hideNavBar: boolean;
    heartbeatToken: number;
    zone: NgZone;
    showUIUpdatedBanner: boolean;
    languageSubscriber: Subscription;
    themeSubscriber: Subscription;
    versionWorkerSubscription: Subscription;
    _routerSubscription: Subscription;
    _routerNavEndSubscription: Subscription;
    displayResolver: boolean;
    toasterConfigDefault: any;
    toasterConfigErrorHTTP: any;
    toasterConfigErrorHTTPLocked: any;
    lastPing: number;
    eventsRouteSubscription: Subscription;
    maintenance: boolean;
    cdsstateSub: Subscription;
    user: AuthentifiedUser;
    previousURL: string

    @ViewChild('gamification')
    eltGamification: ElementRef;
    gameInit: boolean;
    websocket: WebSocketSubject<any>;

    constructor(
        _translate: TranslateService,
        private _language: LanguageStore,
        private _theme: ThemeStore,
        private _activatedRoute: ActivatedRoute,
        private _titleService: Title,
        private _router: Router,
        private _notification: NotificationService,
        private _appService: AppService,
        private _toastService: ToastService,
        private _store: Store,
        private _eventService: EventService,
        private _helpService: HelpService,
        private _ngZone: NgZone,
        private _monitoringService: MonitoringService
    ) {
        this.isAPIAvailable = false;
        this.zone = new NgZone({ enableLongStackTrace: false });
        this.toasterConfigDefault = this._toastService.getConfigDefault();
        this.toasterConfigErrorHTTP = this._toastService.getConfigErrorHTTP();
        this.toasterConfigErrorHTTPLocked = this._toastService.getConfigErrorHTTPLocked();
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
                document.body.className = t;
            } else {
                _theme.set('light');
            }
        });

        this._notification.requestPermission();

        this.eventsRouteSubscription = this._router.events.subscribe(e => {
            if (e instanceof NavigationStart) {
                this.hideNavBar = e.url.startsWith('/auth')
            }
        });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this._monitoringService.getStatus().subscribe(
            (data) => {
                this.isAPIAvailable = true;
                this.load();
            },
            err => {
                this.isAPIAvailable = false;
                setTimeout(() => { window.location.reload() }, 30000);
            }
        );
    }

    load(): void {
        this._helpService.getHelp().subscribe(h => this._store.dispatch(new AddHelp(h)));
        this._store.dispatch(new GetCDSStatus());
        this._store.select(AuthenticationState.user).subscribe(user => {
            if (!user) {
                delete this.user;
                this.isConnected = false;
                this._eventService.stopWebsocket();
            } else {
                this.user = user;
                this.isConnected = true;
                this._eventService.startWebsocket();
            }
        });
        this.startVersionWorker();

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
            .pipe(map((e: NavigationEnd) => {
                if ((!this.previousURL || this.previousURL.split('?')[0] !== e.url.split('?')[0])) {
                    this.previousURL = e.url;
                    this._eventService.subscribeAutoFromUrl(e.url);
                    return;
                }

            }))
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

        this.cdsstateSub = this._store.select(CDSState.getCurrentState()).subscribe(m => {
            // Switch maintenance ON
            if (!this.maintenance && m.maintenance && !this.gameInit && this.isConnected && !this.user.isAdmin()) {
                setTimeout(() => {
                    this.gameInit = true;
                    PACMAN.init(this.eltGamification.nativeElement, '/assets/js/');
                }, 1000);
            }
            this.maintenance = m.maintenance;
        });
    }

    startVersionWorker(): void {
        this._ngZone.runOutsideAngular(() => {
            this.versionWorkerSubscription = Observable.interval(60000)
                .mergeMap(_ => this._monitoringService.getVersion())
                .subscribe(v => {
                    this._ngZone.run(() => {
                        if ((<any>window).cds_version !== v.version) {
                            this.showUIUpdatedBanner = true;
                        }
                    });
                });
        });
    }

    refresh(): void {
        this.zone.runOutsideAngular(() => {
            location.reload(true);
        });
    }
}
