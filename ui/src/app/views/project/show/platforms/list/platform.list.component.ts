import {Component, Input} from '@angular/core';
import {Table} from '../../../../../shared/table/table';
import {Project} from '../../../../../model/project.model';

@Component({
    selector: 'app-project-platform-list',
    templateUrl: './project.platform.list.html',
    styleUrls: ['./project.platform.list.scss']
})
export class ProjectPlatformListComponent extends Table {

    @Input() project: Project;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.project.platforms;
    }
}
