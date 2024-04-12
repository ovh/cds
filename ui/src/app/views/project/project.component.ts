import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';
import { ActivatedRoute } from '@angular/router';
import { Project } from 'app/model/project.model';
import { ProjectStore } from 'app/service/project/project.store';
import { Store } from '@ngxs/store';
import { PreferencesState } from 'app/store/preferences.state';
import * as actionPreferences from "app/store/preferences.action";

@Component({
    selector: 'app-project',
    templateUrl: './project.html',
    styleUrls: ['./project.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectComponent implements OnInit, OnDestroy {
    routeSub: Subscription;
    projSub: Subscription;
    project: Project;
    routerSub: Subscription;
    v2Enabled: boolean = true;

    constructor(
        private _route: ActivatedRoute,
        private _projectStore: ProjectStore,
        private _cd: ChangeDetectorRef,
        private _store: Store
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

        this._store.select(PreferencesState.selectMessageState('ascode-v2')).subscribe(state => {
            this.v2Enabled = !state;
            this._cd.markForCheck();
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	clickCloseBanner(): void {
        this._store.dispatch(new actionPreferences.SaveMessageState({
            messageKey: 'ascode-v2',
            value: true
        }));
	}
}
