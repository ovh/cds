import { Component, Input, OnInit } from '@angular/core';
import {Project} from '../../../../model/project.model';
import {ProjectStore} from '../../../../service/project/project.store';

@Component({
    selector: 'app-project-pipelines',
    templateUrl: './pipeline.list.html',
    styleUrls: ['./pipeline.list.scss']
})
export class ProjectPipelinesComponent implements OnInit {

    @Input() project: Project;

    loading = true;

    constructor(private _projectStore: ProjectStore) {

    }

    ngOnInit() {
      this._projectStore.getProjectPipelinesResolver(this.project.key)
        .finally(() => this.loading = false)
        .subscribe((proj) => this.project = proj);
    }
}
