
import { Component, Input, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { FetchEnvironmentsInProject } from 'app/store/project.action';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Environment } from '../../../../../model/environment.model';
import { Project } from '../../../../../model/project.model';
import { Warning } from '../../../../../model/warning.model';

@Component({
    selector: 'app-environment-list',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss']
})
export class ProjectEnvironmentListComponent implements OnInit, OnDestroy {

    warnMap: Map<string, Array<Warning>>;
    @Input('warnings')
    set warnings(data: Array<Warning>) {
        if (data) {
            this.warnMap = new Map<string, Array<Warning>>();
            data.forEach(w => {
                let arr = this.warnMap.get(w.environment_name);
                if (!arr) {
                    arr = new Array<Warning>();
                }
                arr.push(w);
                this.warnMap.set(w.environment_name, arr);
            });
        }
    }

    @Input() project: Project;
    oldLastModifiedDate: number;
    selectedEnv: Environment;
    selectedEnvIndex = 0;
    envInRoute: string;
    loading: boolean;

    routerSubscription: Subscription;

    constructor(
        private _routerActivatedRoute: ActivatedRoute,
        private _router: Router,
        private store: Store
    ) {
        this.loading = true;
    }

    ngOnDestroy(): void {
        if (this.routerSubscription) {
            this.routerSubscription.unsubscribe();
        }
    }

    ngOnInit(): void {
        let currentTab;

        this.routerSubscription = this._routerActivatedRoute.queryParams.subscribe((q) => {
            if (q['envName']) {
                this.envInRoute = q['envName'];
            }
            currentTab = q['tab'];
            this.selectNewEnv(this.envInRoute, false);
        });

        this.store.dispatch(new FetchEnvironmentsInProject({ projectKey: this.project.key }))
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                if (currentTab !== 'environments') {
                    return;
                }
                if (this.project.environments && this.project.environments.length > 0) {
                    if (this.envInRoute) {
                        this.selectNewEnv(this.envInRoute);
                    } else {
                        this.selectNewEnv(this.project.environments[0].name);
                    }
                }
                this.oldLastModifiedDate = new Date(this.project.last_modified).getTime();
            });
    }

    selectNewEnv(envName: string, updateUrl = true): void {
        if (this.project.environments) {
            let envIndex = this.project.environments.findIndex(e => e.name === this.envInRoute);
            if (envIndex === -1) {
                this.selectedEnvIndex = 0;
            } else {
                this.selectedEnvIndex = envIndex;
            }
        }

        if (updateUrl) {
            this._router.navigate(['/project/', this.project.key], {
                queryParams: { tab: 'environments', envName }
            });
        }
    }

    deleteEnv(): void {
        this.selectedEnvIndex = 0;
    }
}
