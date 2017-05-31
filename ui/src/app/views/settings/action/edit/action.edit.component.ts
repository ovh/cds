import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Action} from '../../../../model/action.model';
import {User} from '../../../../model/user.model';
import {ActionService} from '../../../../service/action/action.service';
import {Subscription} from 'rxjs/Subscription';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';

@Component({
    selector: 'app-action-edit',
    templateUrl: './action.edit.html',
    styleUrls: ['./action.edit.scss']
})
export class ActionEditComponent implements OnInit {
    public ready = true;
    public loadingSave = false;
    public deleteLoading = false;
    public action: Action;
    public isAdmin: boolean;

    private namePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    private actionPatternError = false;

    constructor(private _actionService: ActionService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        if (this._authentificationStore.isConnected()) {
            this.isAdmin = this._authentificationStore.isAdmin();
        }
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            if (params['name'] !== 'add') {
                this.reloadData(params['name']);
            } else {
                this.action = new Action();
                this.ready = true;
            }
        });
    }

    reloadData(name: string): void {
      this._actionService.getAction(name).subscribe( u => {
          this.action = u;
          this.ready = true;
      });
    }

    clickDeleteButton(): void {
      this.deleteLoading = true;
      this._actionService.deleteAction(this.action.name).subscribe( wm => {
          this.deleteLoading = false;
          this._toast.success('', this._translate.instant('action_deleted'));
          this._router.navigate(['../'], { relativeTo: this._route });
      }, () => {
          this.loadingSave = false;
      });
    }

    clickSaveButton(): void {
      if (!this.action.name) {
          return;
      }

      if (!this.namePattern.test(this.action.name)) {
          this.actionPatternError = true;
          return;
      }

      this.loadingSave = true;
      if (this.action.id > 0) {
        this._actionService.updateAction(this.action).subscribe( wm => {
            this.loadingSave = false;
            this._toast.success('', this._translate.instant('action_saved'));
            this.reloadData(this.action.name);
        }, () => {
            this.loadingSave = false;
        });
      }

    }
}
