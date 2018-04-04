import {Component, Input} from '@angular/core';
import {NavbarProjectData} from '../../model/navbar.model';
import {ProjectStore} from '../../service/project/project.store';
import {WorkflowStore} from '../../service/workflow/workflow.store';

@Component({
    selector: 'app-favorite-cards',
    templateUrl: './favorite-cards.component.html',
    styleUrls: ['./favorite-cards.component.scss']
})
export class FavoriteCardsComponent {

    @Input() favorites: Array<NavbarProjectData>;
    @Input() projects: Array<NavbarProjectData>;
    @Input() workflows: Array<NavbarProjectData>;

    newFav = new NavbarProjectData();

    constructor(
      private _projectStore: ProjectStore,
      private _workflowStore: WorkflowStore
    ) { }

    updateFav(fav: NavbarProjectData) {
      switch (fav.type) {
        case 'project':
          this._projectStore.updateFavorite(fav.key)
            .subscribe();
            break;
        case 'workflow':
          this._workflowStore.updateFavorite(fav.key, fav.workflow_name)
            .subscribe();
            break;
      }
      if (this.newFav.type) {
        this.newFav = new NavbarProjectData();
      }
    }
}
