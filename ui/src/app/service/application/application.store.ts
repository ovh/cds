import { Injectable } from '@angular/core';
import { Application } from 'app/model/application.model';
import { NavbarRecentData } from 'app/model/navbar.model';
import * as immutable from 'immutable';
import { BehaviorSubject, Observable } from 'rxjs';


@Injectable()
export class ApplicationStore {

    static RECENT_APPLICATIONS_KEY = 'CDS-RECENT-APPLICATIONS';

    private _recentApplications: BehaviorSubject<immutable.List<NavbarRecentData>> =
        new BehaviorSubject(immutable.List<NavbarRecentData>());


    constructor() {
        this.loadRecentApplication();

    }

    loadRecentApplication(): void {
        let arrayApp = JSON.parse(localStorage.getItem(ApplicationStore.RECENT_APPLICATIONS_KEY));
        this._recentApplications.next(immutable.List.of(...arrayApp));
    }

    /**
     * Get recent application.
     *
     * @returns
     */
    getRecentApplications(): Observable<immutable.List<Application>> {
        return new Observable<immutable.List<Application>>(fn => this._recentApplications.subscribe(fn));
    }

    /**
     * Update recent application viewed.
     *
     * @param key Project unique key
     * @param application Application to add
     */
    updateRecentApplication(key: string, application: Application): void {
        let navbarRecentData = new NavbarRecentData();
        navbarRecentData.project_key = key;
        navbarRecentData.name = application.name;
        let currentRecentApps: Array<NavbarRecentData> = JSON.parse(localStorage.getItem(ApplicationStore.RECENT_APPLICATIONS_KEY));
        if (currentRecentApps) {
            let index: number = currentRecentApps.findIndex(app =>
                app.name === navbarRecentData.name && app.project_key === navbarRecentData.project_key
            );
            if (index >= 0) {
                currentRecentApps.splice(index, 1);
            }
        } else {
            currentRecentApps = new Array<NavbarRecentData>();
        }
        currentRecentApps.splice(0, 0, navbarRecentData);
        currentRecentApps = currentRecentApps.splice(0, 15);
        localStorage.setItem(ApplicationStore.RECENT_APPLICATIONS_KEY, JSON.stringify(currentRecentApps));
        this._recentApplications.next(immutable.List(currentRecentApps));
    }
}
