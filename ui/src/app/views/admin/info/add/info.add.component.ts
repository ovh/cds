import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {Info} from '../../../../model/info.model';
import {InfoService} from '../../../../service/info/info.service';
import {InfoLevelService} from '../../../../shared/info/info.level.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {User} from '../../../../model/user.model';

@Component({
    selector: 'app-info-add',
    templateUrl: './info.add.html',
    styleUrls: ['./info.add.scss']
})
export class InfoAddComponent implements OnInit {
    loading = false;
    deleteLoading = false;
    info: Info;
    currentUser: User;
    canAdd = false;
    private infoLevelsList;

    constructor(private _infoService: InfoService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore, _infoLevelService: InfoLevelService) {
        this.currentUser = this._authentificationStore.getUser();
        this.infoLevelsList = _infoLevelService.getInfoLevels()
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.info = new Info();
        });
    }

    clickSaveButton(): void {
        if (!this.info.title) {
            return;
        }

        this.loading = true;
        this._infoService.createInfo(this.info).subscribe( wm => {
            this.loading = false;
            this._toast.success('', this._translate.instant('info_saved'));
            this._router.navigate(['admin', 'info', this.info.id]);
        }, () => {
            this.loading = false;
        });
    }
}
