import { Component } from '@angular/core';
import { User } from 'app/model/user.model';
import { UserService } from '../../../service/user/user.service';

@Component({
    selector: 'app-auth-reset',
    templateUrl: './reset.html',
    styleUrls: ['./reset.scss'],
})
export class ResetComponent {
    user: User;
    showWaitingMessage = false;

    constructor(
        private _userService: UserService
    ) {
        this.user = new User();
    }

    resetPassword() {
        let bases = document.getElementsByTagName('base');
        let baseHref = null;
        if (bases.length > 0) {
            baseHref = bases[0].href;
        }
        this._userService.resetPassword(this.user, baseHref).subscribe(() => {
            this.showWaitingMessage = true;
        });
    }
}
