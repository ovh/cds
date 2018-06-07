import {Component, DoCheck, Input, OnDestroy, OnInit} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {Environment} from '../../../../../model/environment.model';
import {ProjectStore} from '../../../../../service/project/project.store';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs/Subscription';
import {flatMap} from 'rxjs/operators';
import {Warning} from '../../../../../model/warning.model';

@Component({
    selector: 'app-environment-list',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss']
})
export class ProjectEnvironmentListComponent implements OnInit, DoCheck, OnDestroy {

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

    @Input('project') project: Project;
    oldLastModifiedDate: number;
    selectedEnv: Environment;
    envInRoute: string;
    loading: boolean;

    routerSubscription: Subscription;

    constructor(private _routerActivatedRoute: ActivatedRoute, private _router: Router, private _projectStore: ProjectStore) {
        this.loading = true;
    }

    ngOnDestroy(): void {
        if (this.routerSubscription) {
            this.routerSubscription.unsubscribe();
        }
    }

    ngOnInit(): void {
        let currentTab;
        this.routerSubscription = this._routerActivatedRoute.queryParams
          .map((q) => {
            if (q['envName']) {
                this.envInRoute = q['envName'];
            }
            currentTab = q['tab'];
            return q;
          })
          .pipe(
              flatMap((q) => this._projectStore.getProjectEnvironmentsResolver(this.project.key))
          )
          .subscribe((proj) => {
            if (currentTab !== 'environments') {
              return;
            }
            this.project = proj;
            if (this.project.environments && this.project.environments.length > 0) {
                if (this.envInRoute) {
                    this.selectNewEnv(this.envInRoute);
                } else {
                    this.selectNewEnv(this.project.environments[0].name);
                }
            }
            this.oldLastModifiedDate = new Date(this.project.last_modified).getTime();
            this.loading = false
          }, () => this.loading = false);
    }

    /**
     * Update selected Stage On pipeline update.
     * Do not work with ngOnChange.
     */
    ngDoCheck() {
        if (new Date(this.project.last_modified).getTime() !== this.oldLastModifiedDate) {
            this.oldLastModifiedDate = new Date(this.project.last_modified).getTime();
            // If environment changed - update selected env
            if (this.selectedEnv && this.project.environments) {
                let index = this.project.environments.findIndex(e => e.id === this.selectedEnv.id);
                if (index >= -1) {
                    this.selectedEnv = this.project.environments[index];
                } else {
                    this.selectedEnv = null;
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
                this.selectedEnv = null;
            }
        }
    }

    selectNewEnv(envName): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnv = this.project.environments.find(e => e.name === envName);
            if (!this.selectedEnv) {
                this.selectedEnv = this.project.environments[0];
            }
            this._router.navigate(['/project/', this.project.key], {queryParams: { tab: 'environments', envName: this.selectedEnv.name}});
        }
    }

    deleteEnv(): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnv = this.project.environments[0];
        }
    }
}
