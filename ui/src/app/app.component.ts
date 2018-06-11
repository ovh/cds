import { Title } from '@angular/platform-browser';
import {registerLocaleData} from '@angular/common';
import {Component, OnInit, NgZone} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {AuthentificationStore} from './service/auth/authentification.store';
import {ResolveEnd, ResolveStart, Router, ActivatedRoute, NavigationEnd} from '@angular/router';
import {CDSWorker} from './shared/worker/worker';
import {Subscription} from 'rxjs/Subscription';
import {Observable} from 'rxjs';
import {map, filter, mergeMap} from 'rxjs/operators';
import {LanguageStore} from './service/language/language.store';
import {NotificationService} from './service/notification/notification.service';
import {AutoUnsubscribe} from './shared/decorator/autoUnsubscribe';
import {ToastService} from './shared/toast/ToastService';
import {AppService} from './app.service';
import {LastUpdateService} from './service/sse/lastupdate.sservice';
import {LastModification} from './model/lastupdate.model';
import * as format from 'string-format-obj';
import localeFR from '@angular/common/locales/fr';
import localeEN from '@angular/common/locales/en';

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
@AutoUnsubscribe()
export class AppComponent  implements OnInit {

    open: boolean;
    isConnected = false;
    warningWorker: CDSWorker;
    versionWorker: CDSWorker;
    zone: NgZone;

    currentVersion = 0;
    showUIUpdatedBanner = false;

    warningWorkerSubscription: Subscription;
    languageSubscriber: Subscription;
    versionWorkerSubscription: Subscription;
    _routerSubscription: Subscription;
    _routerNavEndSubscription: Subscription;

    displayResolver = false;
    toasterConfig: any;

    constructor(_translate: TranslateService, private _language: LanguageStore,
                private _activatedRoute: ActivatedRoute, private _titleService: Title,
                private _authStore: AuthentificationStore, private _router: Router,
                private _notification: NotificationService, private _appService: AppService,
                private _last: LastUpdateService, private _toastService: ToastService) {
        this.zone = new NgZone({enableLongStackTrace: false});
        this.toasterConfig = this._toastService.getConfig();
        _translate.addLangs(['en', 'fr']);
        _translate.setDefaultLang('en');
        let browserLang = navigator.language.match(/fr/) ? 'fr' : 'en';
        _translate.use(browserLang.match(/en|fr/) ? browserLang : 'en');
        registerLocaleData(browserLang.match(/fr/) ? localeFR : localeEN);

        this.languageSubscriber = this._language.get().subscribe( l => {
            if (l) {
                _translate.use(l);
            } else {
                _language.set(browserLang.match(/en|fr/) ? browserLang : 'en');
            }
        });

        this._notification.requestPermission();
    }

    ngOnInit(): void {
        this._authStore.getUserlst().subscribe(user => {
            if (!user) {
                this.isConnected = false;
                this.stopWorker(this.warningWorker, this.warningWorkerSubscription);
            } else {
                this.isConnected = true;
                this.startLastUpdateSSE();
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
                return { route, params: Observable.of(params) };
            }))
            .pipe(filter((event) => event.route.outlet === 'primary'))
            .pipe(mergeMap((event) => Observable.zip(event.route.data, event.params)))
            .subscribe((routeData) => {
                if (!Array.isArray(routeData) || routeData.length < 2) {
                    return;
                }
                if (routeData[0]['title']) {
                    let title = format(routeData[0]['title'], routeData[1])
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

    startLastUpdateSSE(): void {
        this._last.getLastUpdate().subscribe(msg => {
            if (msg === 'ACK') {
                return;
            }
            let lastUpdateEvent: LastModification = JSON.parse(msg);
            this._appService.updateCache(lastUpdateEvent);
        });
    }


    startVersionWorker(): void {
        this.stopWorker(this.versionWorker, this.versionWorkerSubscription);
        this.versionWorker = new CDSWorker('./assets/worker/web/version.js');
        this.versionWorker.start({});
        this.versionWorker.response().subscribe( msg => {
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
