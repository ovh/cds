import { registerLocaleData } from '@angular/common';
import { RouterService } from './service/router/router.service';
import {Component, OnInit, NgZone} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {AuthentificationStore} from './service/auth/authentification.store';
import {environment} from '../environments/environment';
import {WarningStore} from './service/warning/warning.store';
import { ResolveEnd, ResolveStart, Router } from '@angular/router';
import {CDSWorker} from './shared/worker/worker';
import {Subscription} from 'rxjs/Subscription';
import {LanguageStore} from './service/language/language.store';
import {NotificationService} from './service/notification/notification.service';
import {AutoUnsubscribe} from './shared/decorator/autoUnsubscribe';
import {AppService} from './app.service';
import {LastUpdateService} from './service/sse/lastupdate.sservice';
import {LastModification} from './model/lastupdate.model';
import localeFR from '@angular/common/locales/fr';
import localeEN from '@angular/common/locales/en';

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
@AutoUnsubscribe([])
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

    displayResolver = false;

    constructor(private _translate: TranslateService, private _language: LanguageStore, private _routerService: RouterService,
                private _authStore: AuthentificationStore, private _warnStore: WarningStore, private _router: Router,
                private _notification: NotificationService, private _appService: AppService, private _last: LastUpdateService) {
        this.zone = new NgZone({enableLongStackTrace: false});
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
                this.startWarningWorker();
            }
            this.startVersionWorker();
        });

        this._router.events.subscribe(e => {
            if (e instanceof ResolveStart) {
                this.displayResolver = true;
            }
            if (e instanceof ResolveEnd) {
                this.displayResolver = false;
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

    /**
     * Start worker to pull warnings.
     * WebWorker for Safari and EDGE
     * SharedWorker for the others  (worker shared between tabs)
     */
    startWarningWorker(): void {
        this.stopWorker(this.warningWorker, this.warningWorkerSubscription);
        this.warningWorker = new CDSWorker('./assets/worker/web/warning.js');
        this.warningWorker.start({
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL});
        this.warningWorker.response().subscribe( msg => {
            if (msg) {
                this.zone.run(() => {
                    this._warnStore.updateWarnings(JSON.parse(msg));
                });
            }
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
