import {Component, OnInit, NgZone} from '@angular/core';
import {TranslateService} from 'ng2-translate';
import {AuthentificationStore} from './service/auth/authentification.store';
import {environment} from '../environments/environment';
import {WarningStore} from './service/warning/warning.store';
import {CDSWorker} from './shared/worker/worker';
import {Subscription} from 'rxjs/Rx';

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
export class AppComponent  implements OnInit {

    open: boolean;
    isConnected = false;
    worker: CDSWorker;
    workerSubscription: Subscription;
    zone: NgZone;

    constructor(private _translate: TranslateService,
                private _authStore: AuthentificationStore, private _warnStore: WarningStore) {
        this.zone = new NgZone({enableLongStackTrace: false});
        _translate.addLangs(['en', 'fr']);
        _translate.setDefaultLang('en');
        let browserLang = _translate.getBrowserLang();
        _translate.use(browserLang.match(/en|fr/) ? browserLang : 'en');
    }

    ngOnInit(): void {
        this._authStore.getUserlst().subscribe(user => {
            if (!user) {
                this.isConnected = false;
                this.stopWarningWorker();
            } else {
                this.isConnected = true;
                this.startWarningWorker();
            }
        });
    }

    /**
     * Stop worker that are pulling warnings.
     */
    stopWarningWorker(): void {
        if (this.worker) {
            this.worker.stop();
        }
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    /**
     * Start worker to pull warnings.
     * WebWorker for Safari and EDGE
     * SharedWorker for the others  (worker shared between tabs)
     */
    startWarningWorker(): void {
        if (this.worker) {
            this.stopWarningWorker();
        }
        this.worker = new CDSWorker('./assets/worker/web/warning.js');
        this.worker.start({ 'user': this._authStore.getUser(), 'session': this._authStore.getSessionToken(), 'api': environment.apiURL});
        this.worker.response().subscribe( msg => {
            if (msg) {
                this.zone.run(() => {
                    this._warnStore.updateWarnings(JSON.parse(msg));
                });
            }
        });
    }
}
