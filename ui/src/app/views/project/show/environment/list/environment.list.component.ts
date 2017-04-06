import {Component, DoCheck, Input, OnDestroy, OnInit} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {Environment} from '../../../../../model/environment.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs/Rx';

@Component({
    selector: 'app-environment-list',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss']
})
export class ProjectEnvironmentListComponent implements OnInit, DoCheck, OnDestroy {

    @Input('project') project: Project;
    oldLastModifiedDate: number;
    selectedEnv: Environment;
    envInRoute: string;

    routerSubscription: Subscription;

    constructor(private _routerActivatedRoute: ActivatedRoute, private _router: Router) {
        this.routerSubscription = this._routerActivatedRoute.queryParams.subscribe(q => {
           if (q['envName']) {
               this.envInRoute = q['envName'];
           }
        });
    }

    ngOnDestroy(): void {
        if (this.routerSubscription) {
            this.routerSubscription.unsubscribe();
        }
    }

    ngOnInit(): void {
        if (this.project.environments && this.project.environments.length > 0) {
            if (this.envInRoute) {
                this.selectNewEnv(this.envInRoute);
            } else {
                this.selectNewEnv(this.project.environments[0].name);
            }
        }
    }

    /**
     * Update selected Stage On pipeline update.
     * Do not work with ngOnChange.
     */
    ngDoCheck() {
        if (this.project.last_modified !== this.oldLastModifiedDate) {
            // If environment changed - update selected env
            if (this.selectedEnv && this.project.environments) {
                let index = this.project.environments.findIndex(e => e.id === this.selectedEnv.id);
                if (index >= -1) {
                    this.selectedEnv = this.project.environments[index];
                } else {
                    this.selectedEnv = undefined;
                }
            } else if (this.project.environments && this.project.environments.length > 0) {
                if (this.envInRoute) {
                    this.selectedEnv = this.project.environments.find(e => {
                        return e.name === this.envInRoute;
                    });
                }
                if (!this.selectedEnv) {
                    this.selectedEnv = this.project.environments[0];
                }
            } else {
                this.selectedEnv = undefined;
            }
        }
    }

    selectNewEnv(envName): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnv = this.project.environments.find(e => e.name === envName);
            this._router.navigate(['/project/', this.project.key], {queryParams: { tab: 'environments', envName: this.selectedEnv.name}});
        }
    }

    deleteEnv(): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnv = this.project.environments[0];
        }
    }
}
