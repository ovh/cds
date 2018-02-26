import {Component, Input, OnInit} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {finalize, first} from 'rxjs/operators';

@Component({
    selector: 'app-project-platforms',
    templateUrl: './project.platforms.html',
    styleUrls: ['./project.platforms.scss']
})
export class ProjectPlatformsComponent implements OnInit {

    @Input() project: Project;
    loading = true;

    constructor(private _projectStore: ProjectStore) { }

    ngOnInit(): void {
        if (this.project.platforms && this.project.platforms.length === 0) {
            this.loading = false;
            return;
        }
        this._projectStore.getProjectPlatformsResolver(this.project.key)
            .pipe(first(), finalize(() => this.loading = false))
            .subscribe((proj) => {
                this.project = proj;
            });
    }
}
