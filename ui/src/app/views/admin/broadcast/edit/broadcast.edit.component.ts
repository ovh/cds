import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthentificationStore} from 'app/service/auth/authentification.store';
import {Broadcast} from 'app/model/broadcast.model';
import {BroadcastService} from 'app/service/broadcast/broadcast.service';
import {BroadcastLevelService} from '../../../../shared/broadcast/broadcast.level.service';
import {SharedService} from '../../../../shared/shared.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {NavbarService} from 'app/service/navbar/navbar.service';
import {TranslateService} from '@ngx-translate/core';
import {User} from 'app/model/user.model';
import {NavbarProjectData} from 'app/model/navbar.model';
import {finalize} from 'rxjs/operators';
import {Subscription} from 'rxjs/Subscription';

@Component({
    selector: 'app-broadcast-edit',
    templateUrl: './broadcast.edit.html',
    styleUrls: ['./broadcast.edit.scss']
})
export class BroadcastEditComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    broadcast: Broadcast;
    currentUser: User;
    canEdit = false;
    private broadcastLevelsList;
    levels = Array<string>();
    projects: Array<NavbarProjectData> = [];
    _navbarSub: Subscription;

    constructor(
        private sharedService: SharedService,
        private _navbarService: NavbarService,
        private _broadcastService: BroadcastService,
        private _toast: ToastService, private _translate: TranslateService,
        private _route: ActivatedRoute, private _router: Router,
        private _authentificationStore: AuthentificationStore, _broadcastLevelService: BroadcastLevelService
    ) {
        this.currentUser = this._authentificationStore.getUser();
        this.broadcastLevelsList = _broadcastLevelService.getBroadcastLevels()
        this.broadcastLevelsList.forEach(element => {
            this.levels.push(element.value);
        });
        this._navbarSub = this._navbarService.getData(true)
        .subscribe((data) => {
            this.loading = false;
            if (Array.isArray(data)) {
                this.projects = data.filter((elt) => elt.type === 'project');
                this.currentUser = this._authentificationStore.getUser();
            }
        });
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            if (params['id'] !== 'add') {
                this.reloadData(params['id']);
            } else {
                this.broadcast = new Broadcast();
            }
        });
    }

    reloadData(broadcastId: string): void {
        this._broadcastService.getBroadcastById(broadcastId).subscribe( broadcast => {
            this.broadcast = broadcast;
            if (this.currentUser.admin) {
                this.canEdit = true;
                return;
            }
        });
    }

    clickDeleteButton(): void {
        this.deleteLoading = true;
        this._broadcastService.deleteBroadcast(this.broadcast)
            .pipe(finalize(() => this.deleteLoading = false))
            .subscribe( wm => {
                this._toast.success('', this._translate.instant('broadcast_deleted'));
                this._router.navigate(['../'], { relativeTo: this._route });
            });
    }

    clickSaveButton(): void {
      if (!this.broadcast.title) {
          return;
      }

      this.loading = true;
      if (this.broadcast.id > 0) {
        this._broadcastService.updateBroadcast(this.broadcast)
            .pipe(finalize(() => this.loading = false))
            .subscribe( broadcast => {
                this._toast.success('', this._translate.instant('broadcast_saved'));
                this._router.navigate(['admin', 'broadcast', this.broadcast.id]);
        });
      } else {
        this._broadcastService.createBroadcast(this.broadcast)
            .pipe(finalize(() => this.loading = false))
            .subscribe( broadcast => {
                this._toast.success('', this._translate.instant('broadcast_saved'));
                this._router.navigate(['admin', 'broadcast', this.broadcast.id]);
        });
      }
    }

    getContentHeight(): number {
        return this.sharedService.getTextAreaheight(this.broadcast.content);
    }
}
