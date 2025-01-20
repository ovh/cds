import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Project } from '../../../model/project.model';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { lastValueFrom } from 'rxjs';
import { ProjectService } from 'app/service/project/project.service';
import { NzMessageService } from 'ng-zorro-antd/message';
import { ErrorUtils } from 'app/shared/error.utils';
import { V2ProjectService } from 'app/service/projectv2/project.service';

@Component({
    selector: 'app-project-list',
    templateUrl: './project.list.component.html',
    styleUrls: ['./project.list.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectListComponent implements OnInit, OnDestroy {
    projects: Array<Project> = [];
    filteredProjects: Array<Project> = [];
    loading = true;

    set filter(filter: string) {
        let filterLower = filter.toLowerCase();
        this.filteredProjects = this.projects.filter((proj) => proj.name.toLowerCase().indexOf(filterLower) !== -1 || proj.key === filter);
    }

    constructor(
        private _projectService: ProjectService,
        private _v2ProjectService: V2ProjectService,
        private _messageService: NzMessageService,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.load();
    }

    async load() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            const projects = await lastValueFrom(this._projectService.getProjects());
            const projectsv2 = await lastValueFrom(this._v2ProjectService.getAll());
            this.projects = [].concat(projects)
                .concat(projectsv2.filter(pv2 => projects.findIndex(p => p.key === pv2.key) === -1));
            this.projects.sort((a, b) => { return a.name < b.name ? -1 : 1; })
            this.filteredProjects = [].concat(this.projects);
        } catch (e: any) {
            this._messageService.error(`Unable to load projects: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }

}
