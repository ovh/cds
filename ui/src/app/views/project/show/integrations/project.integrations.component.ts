import {Component, Input, OnInit} from '@angular/core';
import {finalize, first} from 'rxjs/operators';
import {Project} from '../../../../model/project.model';
import {ProjectStore} from '../../../../service/project/project.store';

@Component({
    selector: 'app-project-integrations',
    templateUrl: './project.integrations.html',
    styleUrls: ['./project.integrations.scss']
})
export class ProjectIntegrationsComponent implements OnInit {

    @Input() project: Project;
    loading = true;

    constructor(private _projectStore: ProjectStore) { }

    ngOnInit(): void {
        if (this.project.integrations && this.project.integrations.length === 0) {
            this.loading = false;
            return;
        }
        this._projectStore.getProjectIntegrationsResolver(this.project.key)
            .pipe(first(), finalize(() => this.loading = false))
            .subscribe((proj) => {
                this.project = proj;
            });
    }
}
