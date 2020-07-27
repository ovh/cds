import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { FetchIntegrationsInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import { Project } from '../../../../model/project.model';

@Component({
    selector: 'app-project-integrations',
    templateUrl: './project.integrations.html',
    styleUrls: ['./project.integrations.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectIntegrationsComponent implements OnInit {

    @Input() project: Project;
    loading = true;

    constructor(private store: Store, private _cd: ChangeDetectorRef) { }

    ngOnInit(): void {
        this.store.dispatch(new FetchIntegrationsInProject({ projectKey: this.project.key }))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }));
    }
}
