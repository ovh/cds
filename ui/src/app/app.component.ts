import {Component, Type, OnInit} from '@angular/core';
import {TranslateService} from 'ng2-translate';
import {AuthentificationStore} from './service/auth/authentification.store';
import {environment} from '../environments/environment';
import {WarningStore} from './service/warning/warning.store';

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss']
})
export class AppComponent extends Type implements OnInit {

    open: boolean;
    isConnected = false;
    warningSharedWorker = null;
    warningWebWorker: Worker = null;

    constructor(private translate: TranslateService,
                private _authStore: AuthentificationStore, private _warnStore: WarningStore) {
        super();

        translate.addLangs(['en', 'fr']);
        translate.setDefaultLang('en');
        let browserLang = translate.getBrowserLang();
        translate.use(browserLang.match(/en|fr/) ? browserLang : 'en');
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
        if (typeof (SharedWorker) !== 'undefined') {
            if (this.warningSharedWorker) {
                this.warningSharedWorker.port.close();
            }
        } else {
            if (this.warningWebWorker) {
                this.warningWebWorker.terminate();
            }
        }
    }

    /**
     * Start worker to pull warnings.
     * WebWorker for Safari and EDGE
     * SharedWorker for the others  (worker shared between tabs)
     */
    startWarningWorker(): void {
        // Run worker to pull CDS Warning
        if (typeof (SharedWorker) !== 'undefined') {
            if (!this.warningSharedWorker) {
                this.warningSharedWorker = new SharedWorker('./assets/worker/shared/warning.js');
                this.warningSharedWorker.port.postMessage({ 'user': this._authStore.getUser(), 'api': environment.apiURL});
                this.warningSharedWorker.port.onmessage = ((e) => {
                    if (e.data && e.data !== 'null') {
                        this._warnStore.updateWarnings(JSON.parse(e.data));
                    }
                });
                this.warningSharedWorker.port.onerror = function (e) {
                    console.log('Warning Worker Error: ', e);
                };
            }
        } else {
            // Use web worker for safari, and edge. Web Workers are not shared between tabs
            if (!this.warningWebWorker) {
                this.warningWebWorker = new Worker('./assets/worker/web/warning.js');
                this.warningWebWorker.postMessage({ 'user': this._authStore.getUser(), 'api': environment.apiURL});
                this.warningWebWorker.onmessage = ((e) => {
                    if (e.data !== 'null') {
                        this._warnStore.updateWarnings(JSON.parse(e.data));
                    }
                });
                this.warningWebWorker.onerror = function (e) {
                    console.log('Warning Worker Error: ', e);
                };
            }
        }
    }
}
