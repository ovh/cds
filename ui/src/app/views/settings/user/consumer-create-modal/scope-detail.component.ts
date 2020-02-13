import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, Output } from '@angular/core';
import { AuthConsumerScopeDetail, AuthConsumerScopeEndpoint } from 'app/model/authentication.model';

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
            this.selectedRouteMethods[endpoint.route] = { ALL: false };
            endpoint.methods.forEach(method => {
                this.selectedRouteMethods[endpoint.route][method] = false;
            });
        });
    }
    @Output() onChange = new EventEmitter<AuthConsumerScopeDetail>();

    advancedOpen: boolean;
    allReadRoutesSelected: boolean;
    allWriteRoutesSelected: boolean;
    allRoutesSelected: boolean;
    selectedRouteMethods: any;

    constructor(private _cd: ChangeDetectorRef) { }

    sendChangeEvent() {
        let scopeDetail = <AuthConsumerScopeDetail>{
            scope: this.scopeDetail.scope
        };

        // If all routes selected returns with empty endpoints list
        if (this.allRoutesSelected) {
            this.onChange.emit(scopeDetail);
            return
        }

        scopeDetail.endpoints = []
        const routes = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < routes.length; i++) {
            let endpoint = <AuthConsumerScopeEndpoint>{
                route: routes[i]
            };

            // If all methods selected returns with empty methods list
            if (this.selectedRouteMethods[routes[i]].ALL) {
                scopeDetail.endpoints.push(endpoint);
                continue;
            }

            endpoint.methods = [];
            const methods = Object.keys(this.selectedRouteMethods[routes[i]]);
            for (let j = 0; j < methods.length; j++) {
                if (methods[j] !== 'ALL' && this.selectedRouteMethods[routes[i]][methods[j]]) {
                    endpoint.methods.push(methods[j]);
                }
            }
            if (endpoint.methods && endpoint.methods.length > 0) {
                scopeDetail.endpoints.push(endpoint);
            }
        }
        if (scopeDetail.endpoints && scopeDetail.endpoints.length > 0) {
            this.onChange.emit(scopeDetail);
        }
    }

    clickAdvanced() {
        this.advancedOpen = !this.advancedOpen;
        this._cd.markForCheck();
    }

    clickSelectAllRoutesRead() {
        this.allReadRoutesSelected = !this.allReadRoutesSelected;

        this.allRoutesSelected = this.allReadRoutesSelected && this.allWriteRoutesSelected;

        const routes = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < routes.length; i++) {
            if (this.selectedRouteMethods[routes[i]].GET != null) {
                this.selectedRouteMethods[routes[i]].GET = this.allReadRoutesSelected;
            }

            this.syncRouteAllCheckbox(routes[i]);
        }

        this._cd.markForCheck();

        this.sendChangeEvent();
    }

    clickSelectAllRoutesWrite() {
        this.allWriteRoutesSelected = !this.allWriteRoutesSelected;

        this.allRoutesSelected = this.allReadRoutesSelected && this.allWriteRoutesSelected;

        const routes = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < routes.length; i++) {
            if (this.selectedRouteMethods[routes[i]].POST != null) {
                this.selectedRouteMethods[routes[i]].POST = this.allWriteRoutesSelected;
            }
            if (this.selectedRouteMethods[routes[i]].PUT != null) {
                this.selectedRouteMethods[routes[i]].PUT = this.allWriteRoutesSelected;
            }
            if (this.selectedRouteMethods[routes[i]].DELETE != null) {
                this.selectedRouteMethods[routes[i]].DELETE = this.allWriteRoutesSelected;
            }

            this.syncRouteAllCheckbox(routes[i]);
        }

        this._cd.markForCheck();

        this.sendChangeEvent();
    }

    clickSelectAllRoutes() {
        this.allRoutesSelected = !this.allRoutesSelected;

        this.allReadRoutesSelected = this.allRoutesSelected;
        this.allWriteRoutesSelected = this.allRoutesSelected;

        // set or unset all routes
        const routes = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < routes.length; i++) {
            Object.keys(this.selectedRouteMethods[routes[i]]).forEach(method => {
                this.selectedRouteMethods[routes[i]][method] = this.allRoutesSelected;
            });
        }

        this._cd.markForCheck();

        this.sendChangeEvent();
    }

    clickMethod(route: string, method: string) {
        if (this.selectedRouteMethods[route]) {
            this.selectedRouteMethods[route][method] = !this.selectedRouteMethods[route][method];

            this.syncRouteAllCheckbox(route);

            this.syncGlobalCheckboxes();

            this._cd.markForCheck();

            this.sendChangeEvent();
        }
    }

    clickAllMethods(route: string) {
        if (this.selectedRouteMethods[route]) {
            this.selectedRouteMethods[route].ALL = !this.selectedRouteMethods[route].ALL;

            // set or unset all methods
            Object.keys(this.selectedRouteMethods[route]).forEach(method => {
                if (method !== 'ALL') {
                    this.selectedRouteMethods[route][method] = this.selectedRouteMethods[route].ALL;
                }
            });

            this.syncGlobalCheckboxes();

            this._cd.markForCheck();

            this.sendChangeEvent();
        }
    }

    syncRouteAllCheckbox(key: string) {
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
    }

    syncGlobalCheckboxes() {
        this.syncGlobalReadCheckbox();
        this.syncGlobalWriteCheckbox();
        this.syncGlobalAllCheckbox();
    }

    syncGlobalReadCheckbox() {
        // check if all write routes are selected
        let allReadRoutesSelected = true;
        const routes = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < routes.length; i++) {
            if (this.selectedRouteMethods[routes[i]].GET != null && !this.selectedRouteMethods[routes[i]].GET) {
                allReadRoutesSelected = false;
                break;
            }
        }
        this.allReadRoutesSelected = allReadRoutesSelected;
    }

    syncGlobalWriteCheckbox() {
        // check if all write routes are selected
        let allWriteRoutesSelected = true;
        const routes = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < routes.length; i++) {
            if (this.selectedRouteMethods[routes[i]].POST != null && !this.selectedRouteMethods[routes[i]].POST) {
                allWriteRoutesSelected = false;
            }
            if (this.selectedRouteMethods[routes[i]].PUT != null && !this.selectedRouteMethods[routes[i]].PUT) {
                allWriteRoutesSelected = false;
            }
            if (this.selectedRouteMethods[routes[i]].DELETE != null && !this.selectedRouteMethods[routes[i]].DELETE) {
                allWriteRoutesSelected = false;
            }
            if (!allWriteRoutesSelected) {
                break;
            }
        }
        this.allWriteRoutesSelected = allWriteRoutesSelected;
    }

    syncGlobalAllCheckbox() {
        // check if all routes are selected
        let allRoutesSelected = true;
        const routes = Object.keys(this.selectedRouteMethods);
        for (let i = 0; i < routes.length; i++) {
            if (!this.selectedRouteMethods[routes[i]].ALL) {
                allRoutesSelected = false;
                break;
            }
        }
        this.allRoutesSelected = allRoutesSelected;
    }
}
