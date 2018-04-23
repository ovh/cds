import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthentificationStore} from 'app/service/auth/authentification.store';
import {Broadcast} from 'app/model/broadcast.model';
import {BroadcastService} from 'app/service/broadcast/broadcast.service';
import {BroadcastLevelService} from '../../../../shared/broadcast/broadcast.level.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {NavbarService} from 'app/service/navbar/navbar.service';
import {TranslateService} from '@ngx-translate/core';
import {User} from 'app/model/user.model';
import {NavbarProjectData} from 'app/model/navbar.model';
import {finalize} from 'rxjs/operators';
import {Subscription} from 'rxjs/Subscription';

@Component({
    selector: 'app-broadcast-add',
    templateUrl: './broadcast.add.html',
    styleUrls: ['./broadcast.add.scss']
})
export class BroadcastAddComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    broadcast: Broadcast;
    currentUser: User;
    canAdd = false;
    private broadcastLevelsList;
    projects: Array<NavbarProjectData> = [];
    _navbarSub: Subscription;

    constructor(
        private _navbarService: NavbarService,
        private _broadcastService: BroadcastService,
        private _toast: ToastService, private _translate: TranslateService,
        private _route: ActivatedRoute, private _router: Router,
        private _authentificationStore: AuthentificationStore, _broadcastLevelService: BroadcastLevelService
    ) {
        this._navbarSub = this._navbarService.getData(true)
        .subscribe((data) => {
        this.loading = false;
        if (Array.isArray(data)) {
            this.projects = data.filter((elt) => elt.type === 'project');
            this.currentUser = this._authentificationStore.getUser();
            this.broadcastLevelsList = _broadcastLevelService.getBroadcastLevels();
        }
        });
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.broadcast = new Broadcast();
        });
    }

    clickSaveButton(): void {
        if (!this.broadcast.title) {
            return;
        }

        this.loading = true;
        this._broadcastService.createBroadcast(this.broadcast)
        .pipe(finalize(() => this.loading = false))
        .subscribe( bc => {
            this._toast.success('', this._translate.instant('broadcast_saved'));
            this._router.navigate(['admin', 'broadcast', bc.id]);
        });
    }
}
