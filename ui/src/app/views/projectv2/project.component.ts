import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';
import { ActivatedRoute } from '@angular/router';
import { Project } from 'app/model/project.model';
import { ProjectStore } from 'app/service/project/project.store';

@Component({
    selector: 'app-projectv2-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ShowComponent implements OnInit, OnDestroy {
    routeSub: Subscription;
    projSub: Subscription;
    project: Project;
    routerSub: Subscription;

    constructor(
        private _route: ActivatedRoute,
        private _projectStore: ProjectStore,
        private _cd: ChangeDetectorRef
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
        this._cd.markForCheck();
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT
}
