import { Component } from '@angular/core';
import { Bookmark } from 'app/model/bookmark.model';
import { NavbarProjectData } from 'app/model/navbar.model';
import { UserService } from 'app/service/services.module';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { NavbarService } from '../../service/navbar/navbar.service';
import { AutoUnsubscribe } from '../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-favorite',
    templateUrl: './favorite.component.html',
    styleUrls: ['./favorite.component.scss']
})
@AutoUnsubscribe()
export class FavoriteComponent {

    favorites: Array<Bookmark> = [];
    projects: Array<NavbarProjectData> = [];
    workflows: Array<NavbarProjectData> = [];
    loading = true;

    _navbarSub: Subscription;

    constructor(
      private _userService: UserService,
      private _navbarService: NavbarService
    ) {
      this.loadBookmarks();

      this._navbarSub = this._navbarService.getData(true)
        .subscribe((data) => {
          this.loading = false;
          if (Array.isArray(data)) {
            let favorites = data.filter((fav) => fav.favorite);
            this.projects = data.filter((elt) => elt.type === 'project');
            this.workflows = data.filter((elt) => {
              return elt.type === 'workflow' &&
                !favorites.find((fav) => fav.type === 'workflow' && fav.workflow_name === elt.workflow_name);
            });
          }
        });
    }

    loadBookmarks() {
      this.loading = true;
      this._userService.getBookmarks()
        .pipe(
          first(),
          finalize(() => this.loading = false)
        ).subscribe((bookmarks) => this.favorites = bookmarks);
    }

    favoriteUpdated() {
      this.loadBookmarks();
    }
}
