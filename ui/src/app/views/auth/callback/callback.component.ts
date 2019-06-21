import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { jws } from 'jsrsasign';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-auth-callback',
    templateUrl: './callback.html',
    styleUrls: ['./callback.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class CallbackComponent implements OnInit {
    paramsSub: Subscription;
    missingParams: boolean;
    showCTL: boolean;
    code: string;
    state: string;
    loading: boolean;

    constructor(
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router
    ) {
        this.loading = true;
    }

    ngOnInit() {
        this.paramsSub = this._route.params.subscribe(params => {
            this.loading = false;
            // let consumerType = params['consumerType'];

            this.code = this._route.snapshot.queryParams.code;
            this.state = this._route.snapshot.queryParams.state;

            if (!this.code || !this.state) {
                this.missingParams = true;
                this._cd.detectChanges();
                return;
            }

            // If the origin is cdsctl, show the code and the state for copy
            let payload = jws.JWS.parse(this.state).payloadObj;
            if (payload.origin === 'cdsctl') {
                this.showCTL = true;
                this._cd.detectChanges();
                return;
            }
        });
    }

    confirmCopy() {
        this._toast.success('', this._translate.instant('auth_value_copied'));
    }

    navigateToSignin() {
        this._router.navigate(['/auth/signin']);
    }
}
