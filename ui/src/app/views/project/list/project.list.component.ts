import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { Subscription } from 'rxjs/Subscription';
import { Project } from '../../../model/project.model';
import { ProjectStore } from '../../../service/project/project.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-project-list',
    templateUrl: './project.list.component.html',
    styleUrls: ['./project.list.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectListComponent implements OnDestroy {
    projects: Array<Project> = [];
    filteredProjects: Array<Project> = [];
    loading = true;

    projectSub: Subscription;

    set filter(filter: string) {
        let filterLower = filter.toLowerCase();
        this.filteredProjects = this.projects.filter((proj) => proj.name.toLowerCase().indexOf(filterLower) !== -1 || proj.key === filter);
    }

    constructor(private _projectStore: ProjectStore, private _cd: ChangeDetectorRef) {
        this.projectSub = this._projectStore.getProjectsList()
            .subscribe((projects) => {
                if (projects) {
                    this.loading = false;
                    this.projects = projects.toArray();
                    this.filteredProjects = projects.toArray();
                    this._cd.markForCheck();
                }
            });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT
}
