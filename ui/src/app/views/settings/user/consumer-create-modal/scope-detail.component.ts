import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { AuthConsumerScopeDetail } from 'app/model/authentication.model';

@Component({
    selector: 'app-scope-detail',
    templateUrl: './scope-detail.html',
    styleUrls: ['./scope-detail.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ScopeDetailComponent {
    scopeDetail: AuthConsumerScopeDetail;
    @Input() set scope(value: AuthConsumerScopeDetail) {
        this.scopeDetail = value;
        this.allRoutesSelected = false;
        this.selectedRouteMethods = {};
        this.scopeDetail.endpoints.forEach(endpoint => {
            const key = this.scopeDetail.scope + '-' + endpoint.route;
            this.selectedRouteMethods[key] = { ALL: false };
            endpoint.methods.forEach(method => {
                this.selectedRouteMethods[key][method] = false;
            });
        });
    }

    advancedOpen: boolean;
    allRoutesSelected: boolean;
    selectedRouteMethods: any;

    constructor(private _cd: ChangeDetectorRef) { }

    clickAdvanced() {
        this.advancedOpen = !this.advancedOpen;
        this._cd.markForCheck();
    }

    clickSelectAllRoutes() {
        this.allRoutesSelected = !this.allRoutesSelected;

        // set or unset all routes
        let keys = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < keys.length; i++) {
            Object.keys(this.selectedRouteMethods[keys[i]]).forEach(method => {
                this.selectedRouteMethods[keys[i]][method] = this.allRoutesSelected;
            });
        }
    }

    clickMethod(key: string, method: string) {
        if (this.selectedRouteMethods[key]) {
            this.selectedRouteMethods[key][method] = !this.selectedRouteMethods[key][method];

            // check if all methods are selected
            let allMethodsSelected = true;
            const methods = Object.keys(this.selectedRouteMethods[key]);
            for (let i = 0; i < methods.length; i++) {
                if (methods[i] !== 'ALL' && !this.selectedRouteMethods[key][methods[i]]) {
                    allMethodsSelected = false;
                    break;
                }
            }
            this.selectedRouteMethods[key].ALL = allMethodsSelected;

            // check if all routes are selected
            let allRoutesSelected = true;
            const keys = Object.keys(this.selectedRouteMethods);
            for (let i = 0; i < keys.length; i++) {
                if (!this.selectedRouteMethods[keys[i]].ALL) {
                    allRoutesSelected = false;
                }
            }
            this.allRoutesSelected = allRoutesSelected;

            this._cd.markForCheck();
        }
    }

    clickAllMethods(key: string) {
        if (this.selectedRouteMethods[key]) {
            this.selectedRouteMethods[key].ALL = !this.selectedRouteMethods[key].ALL;

            // set or unset all methods
            Object.keys(this.selectedRouteMethods[key]).forEach(method => {
                if (method !== 'ALL') {
                    this.selectedRouteMethods[key][method] = this.selectedRouteMethods[key].ALL;
                }
            });

            // check if all routes are selected
            let allRoutesSelected = true;
            const keys = Object.keys(this.selectedRouteMethods);
            for (let i = 0; i < keys.length; i++) {
                if (!this.selectedRouteMethods[keys[i]].ALL) {
                    allRoutesSelected = false;
                }
            }
            this.allRoutesSelected = allRoutesSelected;

            this._cd.markForCheck();
        }
    }
}
