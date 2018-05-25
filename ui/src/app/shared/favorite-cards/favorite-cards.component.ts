import {Component, Input} from '@angular/core';
import {NavbarProjectData} from '../../model/navbar.model';
import {ProjectStore} from '../../service/project/project.store';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-favorite-cards',
    templateUrl: './favorite-cards.component.html',
    styleUrls: ['./favorite-cards.component.scss']
})
export class FavoriteCardsComponent {

    @Input() favorites: Array<NavbarProjectData>;
    @Input() centered = true;
    @Input('projects')
    set projects(projects: Array<NavbarProjectData>) {
         this._projects = projects;
         if (projects) {
             this.filteredProjects = projects.filter((prj) => !this.favorites.find((fav) => fav.type === 'project' && fav.key === prj.key));
         }
    }
    get projects(): Array<NavbarProjectData> {
        return this._projects;
    }
    @Input() workflows: Array<NavbarProjectData>;

    loading = {};
    newFav = new NavbarProjectData();
    filteredProjects: Array<NavbarProjectData> = [];
    filteredWf: Array<NavbarProjectData> = [];
    set projectKeySelected(projectKey: string) {
        this._projectKeySelected = projectKey;
        if (projectKey) {
            this.filteredWf = this.workflows.filter((wf) => wf.key === projectKey);
        }
    }
    get projectKeySelected(): string {
        return this._projectKeySelected;
    }

    private _projectKeySelected: string;
    private _projects: Array<NavbarProjectData> = [];

    constructor(
        private _projectStore: ProjectStore,
        private _workflowStore: WorkflowStore
    ) { }

    updateFav(fav: NavbarProjectData) {
        let key = fav.key + fav.workflow_name;
        if (this.loading[key]) {
            return;
        }
        this.loading[key] = true;
        switch (fav.type) {
            case 'project':
                this._projectStore.updateFavorite(fav.key)
                    .pipe(finalize(() => {
                        this.loading[key] = false;
                        this.newFav = new NavbarProjectData();
                        this.projectKeySelected = null;
                    }))
                    .subscribe();
            break;
            case 'workflow':
                this._workflowStore.updateFavorite(fav.key, fav.workflow_name)
                    .pipe(finalize(() => {
                        this.loading[key] = false;
                        this.newFav = new NavbarProjectData();
                        this.projectKeySelected = null;
                    }))
                    .subscribe();
            break;
            default:
                this.newFav = new NavbarProjectData();
                this.projectKeySelected = null;
        }
    }
}
