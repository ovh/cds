import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {Info} from '../../../../model/info.model';
import {InfoService} from '../../../../service/info/info.service';
import {InfoLevelService} from '../../../../shared/info/info.level.service';
import {SharedService} from '../../../../shared/shared.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {User} from '../../../../model/user.model';

@Component({
    selector: 'app-info-edit',
    templateUrl: './info.edit.html',
    styleUrls: ['./info.edit.scss']
})
export class InfoEditComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    info: Info;
    currentUser: User;
    canEdit = false;
    private infoLevelsList;

    constructor(private sharedService: SharedService,
                private _infoService: InfoService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore, _infoLevelService: InfoLevelService) {
        this.currentUser = this._authentificationStore.getUser();
        this.infoLevelsList = _infoLevelService.getInfoLevels()
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            if (params['id'] !== 'add') {
                this.reloadData(params['id']);
            } else {
                this.info = new Info();
            }
        });
    }

    reloadData(infoId: string): void {
        this._infoService.getInfoById(infoId).subscribe( info => {
            this.info = info;
            if (this.currentUser.admin) {
                this.canEdit = true;
                return;
            }
        });
    }

    clickDeleteButton(): void {
        this.deleteLoading = true;
        this._infoService.deleteInfo(this.info).subscribe( wm => {
            this.deleteLoading = false;
            this._toast.success('', this._translate.instant('info_deleted'));
            this._router.navigate(['../'], { relativeTo: this._route });
        }, () => {
            this.loading = false;
        });
    }

    clickSaveButton(): void {
      if (!this.info.title) {
          return;
      }

      this.loading = true;
      if (this.info.id > 0) {
        this._infoService.updateInfo(this.info).subscribe( info => {
            this.loading = false;
            this._toast.success('', this._translate.instant('info_saved'));
            this._router.navigate(['admin', 'info', this.info.id]);
        }, () => {
            this.loading = false;
        });
      } else {
        this._infoService.createInfo(this.info).subscribe( info => {
            this.loading = false;
            this._toast.success('', this._translate.instant('info_saved'));
            this._router.navigate(['admin', 'info', this.info.id]);
        }, () => {
            this.loading = false;
        });
      }
    }

    getMessageHeight(): number {
        return this.sharedService.getTextAreaheight(this.info.message);
    }
}
