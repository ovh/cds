import {Component} from '@angular/core';
import {Broadcast} from 'app/model/broadcast.model';
import {NavbarProjectData} from 'app/model/navbar.model';
import {Subscription} from 'rxjs';
import {User} from '../../model/user.model';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {BroadcastService} from '../../service/broadcast/broadcast.service';
import {NavbarService} from '../../service/navbar/navbar.service';
import {ProjectStore} from '../../service/project/project.store';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-home',
    templateUrl: './home.html',
    styleUrls: ['./home.scss']
})
@AutoUnsubscribe()
export class HomeComponent {

    favorites: Array<NavbarProjectData> = [];
    broadcasts: Array<Broadcast> = [];
    loading = true;
    loadingBroadcasts = true;
    user: User;

    _navbarSub: Subscription;
    _broadcastSub: Subscription;

    constructor(
        private _navbarService: NavbarService,
        private _projectStore: ProjectStore,
        private _workflowStore: WorkflowStore,
        private _broadcastService: BroadcastService,
        private _authStore: AuthentificationStore
    ) {
        this.user = this._authStore.getUser();
        this._navbarSub = this._navbarService.getData(true)
            .subscribe((data) => {
                this.loading = false;
                if (Array.isArray(data)) {
                    this.favorites = data.filter((fav) => fav.favorite);
                }
            });

        this._broadcastSub = this._broadcastService.getBroadcastsListener()
            .subscribe((broadcasts) => {
                this.loadingBroadcasts = false
                if (Array.isArray(broadcasts)) {
                    this.broadcasts = broadcasts.filter((br) => !br.read && !br.archived).slice(0, 5);
                }
            }, () => this.loadingBroadcasts = false);
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

    markAsRead(id: number) {
        for (let i = 0; i < this.broadcasts.length; i++) {
            if (this.broadcasts[i].id === id) {
                this.broadcasts[i].updating = true;
            }
        }
        this._broadcastService.markAsRead(id)
            .subscribe();
    }
}
