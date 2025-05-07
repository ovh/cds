import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';
import { Project } from 'app/model/project.model';
import { Store } from '@ngxs/store';
import { ProjectState } from 'app/store/project.state';
import { ProjectV2State } from 'app/store/project-v2.state';

@Component({
    selector: 'app-project',
    templateUrl: './project.html',
    styleUrls: ['./project.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectComponent implements OnInit, OnDestroy {
    projectSub: Subscription;
    projectv2Sub: Subscription;
    project: Project;
    projectv2: Project;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.projectSub = this._store.select(ProjectState.projectSnapshot).subscribe(p => {
            if (!p) { return; }
            this.project = p;
            this._cd.markForCheck();
        });
        this.projectv2Sub = this._store.select(ProjectV2State.current).subscribe(p => {
            if (!p) { return; }
            this.project = p;
            this._cd.markForCheck();
        });
    }
}
