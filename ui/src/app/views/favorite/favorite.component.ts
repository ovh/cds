import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { Bookmark } from 'app/model/bookmark.model';
import { NavbarProjectData } from 'app/model/navbar.model';
import { UserService } from 'app/service/user/user.service';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { NavbarService } from '../../service/navbar/navbar.service';
import { AutoUnsubscribe } from '../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-favorite',
    templateUrl: './favorite.component.html',
    styleUrls: ['./favorite.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class FavoriteComponent implements OnDestroy {

    favorites: Array<Bookmark> = [];
    projects: Array<NavbarProjectData> = [];
    workflows: Array<NavbarProjectData> = [];
    loading = true;

    _navbarSub: Subscription;

    constructor(
      private _userService: UserService,
      private _navbarService: NavbarService,
      private _cd: ChangeDetectorRef
    ) {
      this.loadBookmarks();

      this._navbarSub = this._navbarService.getObservable()
        .subscribe((data) => {
            this._cd.markForCheck();
            this.loading = false;
            if (Array.isArray(data)) {
                let favorites = data.filter((fav) => fav.favorite);
                this.projects = data.filter((elt) => elt.type === 'project');
                this.workflows = data.filter((elt) => elt.type === 'workflow' &&
                    !favorites.find((fav) => fav.type === 'workflow' && fav.workflow_name === elt.workflow_name));
            }
        });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    loadBookmarks() {
      this.loading = true;
      this._userService.getBookmarks()
        .pipe(
          first(),
          finalize(() => {
              this.loading = false;
              this._cd.markForCheck();
          })
        ).subscribe((bookmarks) => this.favorites = bookmarks);
    }

    favoriteUpdated() {
      this.loadBookmarks();
    }
}
