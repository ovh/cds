import { registerLocaleData } from '@angular/common';
import localeEN from '@angular/common/locales/en';
import { Component, NgZone, OnDestroy, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, NavigationEnd, NavigationStart, ResolveEnd, ResolveStart, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { EventService } from 'app/event.service';
import { GetCDSStatus } from 'app/store/cds.action';
import { CDSState } from 'app/store/cds.state';
import { interval, of, zip } from 'rxjs';
import { WebSocketSubject } from 'rxjs/internal-compatibility';
import { concatMap, filter, map, mergeMap } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import format from 'string-format-obj';
import { AppService } from './app.service';
import { AuthSummary } from './model/user.model';
import { NotificationService } from './service/notification/notification.service';
import { HelpService, MonitoringService } from './service/services.module';
import { AutoUnsubscribe } from './shared/decorator/autoUnsubscribe';
import { ToastService } from './shared/toast/ToastService';
import { AuthenticationState } from './store/authentication.state';
import { AddHelp } from './store/help.action';
import { NzNotificationService } from 'ng-zorro-antd/notification';
import { NzConfigService } from "ng-zorro-antd/core/config";
import { CodeEditorConfig } from "ng-zorro-antd/core/config/config";
import { PreferencesState } from './store/preferences.state';
import { Editor } from './model/editor.model';

declare const monaco: any;

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
@AutoUnsubscribe()
export class AppComponent implements OnInit, OnDestroy {

    @ViewChild('templateRootToast') toastTemplate: TemplateRef<any>;
    toastSubs: Subscription;

    open: boolean;
    isAPIAvailable: boolean;
    isConnected: boolean;
    hideNavBar: boolean;
    heartbeatToken: number;
    zone: NgZone;
    showUIUpdatedBanner: boolean;
    themeSubscriber: Subscription;
    versionWorkerSubscription: Subscription;
    _routerSubscription: Subscription;
    _routerNavEndSubscription: Subscription;
    displayResolver: boolean;

    lastPing: number;
    eventsRouteSubscription: Subscription;
    maintenance: boolean;
    cdsstateSub: Subscription;
    currentAuthSummary: AuthSummary;
    previousURL: string;
    websocket: WebSocketSubject<any>;
    loading = true;

    constructor(
        private _translate: TranslateService,
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
        private _monitoringService: MonitoringService,
        private _nzNotificationService: NzNotificationService,
        private _configService: NzConfigService,
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
        this._translate.addLangs(['en']);
        this._translate.setDefaultLang('en');
        this._translate.use('en');
        registerLocaleData(localeEN);

        this.themeSubscriber = this._store.select(PreferencesState.theme).subscribe(t => {
            document.body.className = t;

            if (t === 'night') {
                const style = document.createElement('link');
                style.type = 'text/css';
                style.rel = 'stylesheet';
                style.id = 'dark-theme';
                style.href = 'ngzorro.dark.css';
                document.body.appendChild(style);
            } else {
                const dom = document.getElementById('dark-theme');
                if (dom) {
                    dom.remove();
                }
            }

            const defaultEditorOption: CodeEditorConfig = this._configService.getConfigForComponent('codeEditor')?.defaultEditorOption || {};
            this._configService.set('codeEditor', {
                defaultEditorOption: {
                    ...defaultEditorOption,
                    theme: t === 'night' ? 'vs-dark' : 'vs'
                }
            });
        });

        this._notification.requestPermission();

        this.eventsRouteSubscription = this._router.events.subscribe(e => {
            if (e instanceof NavigationStart) {
                this.hideNavBar = e.url.startsWith('/auth');
            }
        });


    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this._monitoringService.getStatus().subscribe(
            (data) => {
                this.isAPIAvailable = true;
                this.loading = false;
                this.load();
            },
            err => {
                this.isAPIAvailable = false;
                this.loading = false;
                setTimeout(() => {
                    window.location.reload();
                }, 30000);
            }
        );

        this.toastSubs = this._toastService.getObservable().subscribe(data => {
            if (!this.toastTemplate) {
                return;
            }
            this._nzNotificationService.template(this.toastTemplate, data)
        });

        const config: CodeEditorConfig = this._configService.getConfigForComponent('codeEditor');
        this._configService.set('codeEditor', {
            ...config,
            onLoad: () => { monaco.languages.registerCompletionItemProvider("yaml", Editor.completionProvider(monaco)) }
        });
    }

    load(): void {
        this._helpService.getHelp().subscribe(h => this._store.dispatch(new AddHelp(h)));
        this._store.dispatch(new GetCDSStatus());
        this._store.select(AuthenticationState.summary).subscribe(s => {
            if (!s) {
                this.currentAuthSummary = null;
                this.isConnected = false;
                this._eventService.stopWebsocket();
            } else {
                this.currentAuthSummary = s;
                this.isConnected = true;
                localStorage.setItem('CDS-USER', this.currentAuthSummary.user.username);
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
                return { route, params: of(params) };
            }))
            .pipe(filter((event) => event.route.outlet === 'primary'))
            .pipe(mergeMap((event) => zip(event.route.data, event.params)))
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
            this.maintenance = m.maintenance;
        });
    }

    startVersionWorker(): void {
        this._ngZone.runOutsideAngular(() => {
            this.versionWorkerSubscription = interval(60000)
                .pipe(concatMap(_ => this._monitoringService.getVersion()))
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
            location.reload();
        });
    }
}
