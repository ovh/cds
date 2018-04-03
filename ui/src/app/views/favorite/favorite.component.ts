import {Component} from '@angular/core';
import {NavbarService} from '../../service/navbar/navbar.service';
import {ProjectStore} from '../../service/project/project.store';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {NavbarProjectData} from 'app/model/navbar.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-favorite',
    templateUrl: './favorite.component.html',
    styleUrls: ['./favorite.component.scss']
})
@AutoUnsubscribe()
export class FavoriteComponent {

    favorites: Array<NavbarProjectData> = [];
    loading = true;

    _navbarSub: Subscription;

    constructor(
      private _navbarService: NavbarService,
      private _projectStore: ProjectStore,
      private _workflowStore: WorkflowStore,
    ) {
      this._navbarSub = this._navbarService.getData(true)
        .subscribe((data) => {
          this.loading = false;
          if (Array.isArray(data)) {
            this.favorites = data.filter((fav) => fav.favorite);
          }
        });
    }

    deleteFav(fav: NavbarProjectData) {
      switch (fav.type) {
        case 'project':
          this._projectStore.updateFavorite(fav.key)
            .subscribe();
            break;
        case 'workflow':
          this._workflowStore.updateFavorite(fav.key, fav.name)
            .subscribe();
            break;
      }
    }
}
