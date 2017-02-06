import {Component, Input, OnInit} from '@angular/core';
import {Project} from '../../../../../model/project.model';

@Component({
    selector: 'app-environment-list',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss']
})
export class ProjectEnvironmentListComponent implements OnInit {

    @Input('project') project: Project;

    selectedEnvIndex: number;

    constructor() { }

    ngOnInit(): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnvIndex = 0;
        }
    }
}
