import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { Store } from '@ngxs/store';
import { Bookmark } from 'app/model/bookmark.model';
import { NavbarProjectData } from 'app/model/navbar.model';
import { UpdateFavoriteProject } from 'app/store/project.action';
import { UpdateFavoriteWorkflow } from 'app/store/workflow.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-favorite-cards',
    templateUrl: './favorite-cards.component.html',
    styleUrls: ['./favorite-cards.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class FavoriteCardsComponent {

    @Input() favorites: Array<Bookmark>;
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

    @Output() updated: EventEmitter<NavbarProjectData> = new EventEmitter<NavbarProjectData>();

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

    constructor(private store: Store) { }

    updateFav(fav: NavbarProjectData) {
        if (!fav || !fav.type) {
            return;
        }
        let key = fav.key + fav.workflow_name;
        if (this.loading[key]) {
            return;
        }
        this.loading[key] = true;
        switch (fav.type) {
            case 'project':
                this.store.dispatch(new UpdateFavoriteProject({
                    projectKey: fav.key
                })).pipe(finalize(() => {
                    this.loading[key] = false;
                    this.newFav = new NavbarProjectData();
                    this.projectKeySelected = null;
                })).subscribe(() => this.updated.emit(fav));
                break;
            case 'workflow':
                this.store.dispatch(new UpdateFavoriteWorkflow({
                    projectKey: fav.key,
                    workflowName: fav.workflow_name
                })).pipe(finalize(() => {
                    this.loading[key] = false;
                    this.newFav = new NavbarProjectData();
                    this.projectKeySelected = null;
                })).subscribe(() => this.updated.emit(fav));
                break;
            default:
                this.newFav = new NavbarProjectData();
                this.projectKeySelected = null;
        }
    }
}
