import {Component} from '@angular/core';
import {NavbarProjectData} from 'app/model/navbar.model';
import {Subscription} from 'rxjs';
import {NavbarService} from '../../service/navbar/navbar.service';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-favorite',
    templateUrl: './favorite.component.html',
    styleUrls: ['./favorite.component.scss']
})
@AutoUnsubscribe()
export class FavoriteComponent {

    favorites: Array<NavbarProjectData> = [];
    projects: Array<NavbarProjectData> = [];
    workflows: Array<NavbarProjectData> = [];
    loading = true;

    _navbarSub: Subscription;

    constructor(
      private _navbarService: NavbarService
    ) {
      this._navbarSub = this._navbarService.getData(true)
        .subscribe((data) => {
          this.loading = false;
          if (Array.isArray(data)) {
            this.favorites = data.filter((fav) => fav.favorite);
            this.projects = data.filter((elt) => elt.type === 'project');
            this.workflows = data.filter((elt) => {
              return elt.type === 'workflow' &&
                !this.favorites.find((fav) => fav.type === 'workflow' && fav.workflow_name === elt.workflow_name);
            });
          }
        });
    }
}
