import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { Project } from 'app/model/project.model';
import { ProjectService } from 'app/service/project/project.service';
import { Router } from '@angular/router';

@Component({
    selector: 'app-projectv2-top-menu',
    templateUrl: './project.top.menu.html',
    styleUrls: ['./project.top.menu.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectV2TopMenuComponent {

    _currentProject: Project;
    get project(): Project {
        return this._currentProject;
    }
    @Input() set project(data: Project) {
        this._currentProject = data;
        if (data) {
            this.selectedProjectKey = data.key;
        }
    }

    public selectedProjectKey: string;
    public projectList: Array<Project> = new Array<Project>();

    constructor(private _projectService: ProjectService, private _cd: ChangeDetectorRef, private _router: Router) {
        this._projectService.getProjects().subscribe(projects => {
            this.projectList = projects;
            this._cd.markForCheck();
        })
    }

    selectProject(projectKey: string): void {
        this._router.navigate(['/', 'projectv2', projectKey]);
    }
}
