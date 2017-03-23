import {Component, Input, OnInit} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {Environment} from '../../../../../model/environment.model';

@Component({
    selector: 'app-environment-list',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss']
})
export class ProjectEnvironmentListComponent implements OnInit {

    @Input('project') project: Project;

    selectedEnv: Environment;

    constructor() { }

    ngOnInit(): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnv = this.project.environments[0];
        }
    }

    selectNewEnv(envName): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnv = this.project.environments.find(e => e.name === envName);
        }
    }

    deleteEnv(): void {
        if (this.project.environments && this.project.environments.length > 0) {
            this.selectedEnv = this.project.environments[0];
        }
    }
}
