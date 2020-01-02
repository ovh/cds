import {
    ChangeDetectionStrategy,
    Component,
    Input,
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { AuthConsumer } from 'app/model/authentication.model';
import { ToastService } from 'app/shared/toast/ToastService';

@Component({
    selector: 'app-consumer-display-signin-token',
    templateUrl: './consumer-display-signin-token.html',
    styleUrls: ['./consumer-display-signin-token.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConsumerDisplaySigninTokenComponent {
    @Input() signinToken: string;
    @Input() consumer: AuthConsumer;

    constructor(
        private _toast: ToastService,
        private _translate: TranslateService
    ) {}

    confirmCopy() {
        this._toast.success('', this._translate.instant('auth_value_copied'));
    }
}
