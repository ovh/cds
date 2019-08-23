import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Broadcast } from 'app/model/broadcast.model';
import { NavbarProjectData } from 'app/model/navbar.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { BroadcastService } from 'app/service/broadcast/broadcast.service';
import { BroadcastStore } from 'app/service/broadcast/broadcast.store';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { SharedService } from 'app/shared/shared.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-broadcast-edit',
    templateUrl: './broadcast.edit.html',
    styleUrls: ['./broadcast.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class BroadcastEditComponent {
    loading = false;
    deleteLoading = false;
    broadcast: Broadcast;
    broadcastSub: Subscription;
    currentUser: AuthentifiedUser;
    canEdit = false;
    broadcastLevelsList: any;
    levels = Array<string>();
    projects: Array<NavbarProjectData> = [];
    navbarSub: Subscription;
    path: Array<PathItem>;
    paramsSub: Subscription;

    constructor(
        private sharedService: SharedService,
        private _navbarService: NavbarService,
        private _broadcastStore: BroadcastStore,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _store: Store,
        private _broadcastService: BroadcastService,
        private _cd: ChangeDetectorRef
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
        this.broadcastLevelsList = this._broadcastService.getBroadcastLevels()
        this.broadcastLevelsList.forEach(element => {
            this.levels.push(element.value);
        });
        this.navbarSub = this._navbarService.getData(true)
            .subscribe((data) => {
                this.loading = false;
                if (Array.isArray(data)) {
                    let voidProj = new NavbarProjectData();
                    voidProj.type = 'project';
                    voidProj.name = ' ';
                    this.projects = [voidProj].concat(data.filter((elt) => elt.type === 'project'));
                    this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
                }
                this._cd.markForCheck();
            });

        this.paramsSub = this._route.params.subscribe(params => {
            let id = parseInt(params['id'], 10)
            this.broadcastSub = this._broadcastStore.getBroadcasts(id).subscribe(bcs => {
                let broadcast = bcs.get(id)
                if (broadcast) {
                    this.broadcast = broadcast;
                    this.canEdit = this.currentUser.isAdmin();
                    this.updatePath();
                }
            });
            this._cd.markForCheck();
        });
    }

    clickDeleteButton(): void {
        this.deleteLoading = true;
        this._broadcastStore.delete(this.broadcast)
            .pipe(finalize(() => {
                this.deleteLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('broadcast_deleted'));
                this._router.navigate(['admin', 'broadcast']);
            });
    }

    clickSaveButton(): void {
        this.loading = true;
        this._broadcastStore.update(this.broadcast)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(broadcast => {
                this._toast.success('', this._translate.instant('broadcast_saved'));
                this._router.navigate(['admin', 'broadcast', this.broadcast.id]);
            });
    }

    getContentHeight(): number {
        return this.sharedService.getTextAreaheight(this.broadcast.content);
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'broadcast_list_title',
            routerLink: ['/', 'admin', 'broadcast']
        }];

        if (this.broadcast && this.broadcast.id) {
            this.path.push(<PathItem>{
                text: this.broadcast.id + '',
                routerLink: ['/', 'admin', 'broadcast', this.broadcast.id]
            });
        }
    }
}
