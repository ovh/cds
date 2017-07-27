import {Component, OnInit, NgZone} from '@angular/core';
import {TranslateService} from 'ng2-translate';
import {AuthentificationStore} from './service/auth/authentification.store';
import {environment} from '../environments/environment';
import {WarningStore} from './service/warning/warning.store';
import {CDSWorker} from './shared/worker/worker';
import {Subscription} from 'rxjs/Subscription';
import {LanguageStore} from './service/language/language.store';
import {NotificationService} from './service/notification/notification.service';
import {AutoUnsubscribe} from './shared/decorator/autoUnsubscribe';
import {AppService} from './app.service';
import {LastUpdateService} from './service/sse/lastupdate.sservice';
import {LastModification} from './model/lastupdate.model';

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

    constructor(private _translate: TranslateService, private _language: LanguageStore,
                private _authStore: AuthentificationStore, private _warnStore: WarningStore,
                private _notification: NotificationService, private _appService: AppService, private _last: LastUpdateService) {
        this.zone = new NgZone({enableLongStackTrace: false});
        _translate.addLangs(['en', 'fr']);
        _translate.setDefaultLang('en');
        let browserLang = _translate.getBrowserLang();
        _translate.use(browserLang.match(/en|fr/) ? browserLang : 'en');

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
                    let versionJSON = JSON.parse(msg).version;
                    if (this.currentVersion === 0) {
                        this.currentVersion = JSON.parse(msg).version;
                    }

                    if (this.currentVersion < versionJSON.version) {
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
