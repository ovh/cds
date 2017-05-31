import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {User} from '../../../../model/user.model';
import {UserService} from '../../../../service/user/user.service';
import {Subscription} from 'rxjs/Subscription';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';

@Component({
    selector: 'app-user-edit',
    templateUrl: './user.edit.html',
    styleUrls: ['./user.edit.scss']
})
export class UserEditComponent implements OnInit {
    public ready = true;
    public loadingSave = false;
    public deleteLoading = false;
    public user: User;
    private username: string;
    public currentUser: User;

    private usernamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    private userPatternError = false;

    constructor(private _userService: UserService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        this.currentUser = this._authentificationStore.getUser();
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this.username = params['username'];
            this.reloadData();
        });
    }

    reloadData(): void {
      this._userService.getUser(this.username).subscribe( u => {
          this.user = u;
          this.username = this.user.username;
          this.ready = true;
      });
    }

    clickDeleteButton(): void {
      this.deleteLoading = true;
      this._userService.deleteUser(this.user.username).subscribe( wm => {
          this.deleteLoading = false;
          this._toast.success('', this._translate.instant('user_deleted'));
          this._router.navigate(['../'], { relativeTo: this._route });
      }, () => {
          this.loadingSave = false;
      });
    }

    clickSaveButton(): void {
      if (!this.user.username) {
          return;
      }

      if (!this.usernamePattern.test(this.user.username)) {
          this.userPatternError = true;
          return;
      }

      this.loadingSave = true;
      if (this.user.id > 0) {
        this._userService.updateUser(this.username, this.user).subscribe( wm => {
            this.loadingSave = false;
            this._toast.success('', this._translate.instant('user_saved'));
            this._router.navigate(['/settings', 'user', this.user.username], { relativeTo: this._route });
        }, () => {
            this.loadingSave = false;
        });
      }

    }
}
