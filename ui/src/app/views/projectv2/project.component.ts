import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';
import { Store } from '@ngxs/store';
import { ActivatedRoute } from '@angular/router';
import { Project } from 'app/model/project.model';
import { ProjectStore } from 'app/service/project/project.store';
import { PreferencesState } from 'app/store/preferences.state';
import * as actionPreferences from 'app/store/preferences.action';

@Component({
    selector: 'app-projectv2-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ShowComponent implements OnInit, OnDestroy {
    static PANEL_KEY = 'project-v2-sidebar';

    routeSub: Subscription;
    projSub: Subscription;
    project: Project;
    panelSize: number;

    constructor(
        private _store: Store,
        private _route: ActivatedRoute,
        private _projectStore: ProjectStore,
        private _cd: ChangeDetectorRef,
    ) { }

    ngOnInit(): void {
        this.routeSub = this._route.params.subscribe(r => {
            let projectKey = r['key'];
            if (this.projSub) {
                this.projSub.unsubscribe();
            }
            this.projSub = this._projectStore.getProjects(projectKey).subscribe((projCache) => {
                let proj = projCache.get(projectKey);
                if (proj) {
                    this.project = proj;
                    this._cd.markForCheck();
                }
            });
        });


        this.routeSub = this._route.params.subscribe(r => {
            let projectKey = r['key'];
            if (this.projSub) {
                this.projSub.unsubscribe();
            }
            this.projSub = this._projectStore.getProjects(projectKey).subscribe((projCache) => {
                let proj = projCache.get(projectKey);
                if (proj) {
                    this.project = proj;
                    this._cd.markForCheck();
                }
            });
        });

        this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2ShowComponent.PANEL_KEY));
        this._cd.markForCheck();
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    panelEndResize(size: number): void {
        this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: ProjectV2ShowComponent.PANEL_KEY, size: size }));
    }
}
