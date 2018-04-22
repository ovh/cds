import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {Broadcast} from '../../../../model/broadcast.model';
import {BroadcastService} from '../../../../service/broadcast/broadcast.service';
import {BroadcastLevelService} from '../../../../shared/broadcast/broadcast.level.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {User} from '../../../../model/user.model';
import {finalize} from 'rxjs/operators';

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

    constructor(private _broadcastService: BroadcastService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore, _broadcastLevelService: BroadcastLevelService) {
        this.currentUser = this._authentificationStore.getUser();
        this.broadcastLevelsList = _broadcastLevelService.getBroadcastLevels()
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
