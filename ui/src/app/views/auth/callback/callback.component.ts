import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { AuthenticationService } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { jws } from 'jsrsasign';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

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
    loadingSignin: boolean;
    showErrorMessage: boolean;
    showInitTokenForm: boolean;
    consumerType: string;
    payloadData: any;

    constructor(
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _authenticationService: AuthenticationService
    ) {
        this.loading = true;
    }

    ngOnInit() {
        this.paramsSub = this._route.params.subscribe(params => {
            this.consumerType = params['consumerType'];

            this.code = this._route.snapshot.queryParams.code || this._route.snapshot.queryParams.token;
            this.state = this._route.snapshot.queryParams.state || this._route.snapshot.queryParams.request;

            if (!this.code || !this.state) {
                this.loading = false;
                this.missingParams = true;
                this._cd.markForCheck();
                return;
            }

            // If the origin is cdsctl, show the code and the state for copy
            let payload = jws.JWS.parse(this.state).payloadObj;
            if (payload.data) {
                this.payloadData = JSON.parse(payload.data);
            }
            if (this.payloadData && this.payloadData.origin === 'cdsctl') {
                this.loading = false;
                this.showCTL = true;
                this._cd.markForCheck();
                return;
            }

            // If the first connection flag is set, show init token form
            if (this.payloadData && this.payloadData.is_first_connection) {
                this.loading = false;
                this.showInitTokenForm = true;
                this._cd.markForCheck();
                return;
            }

            this.sendSigninRequest();
        });
    }

    confirmCopy() {
        this._toast.success('', this._translate.instant('auth_value_copied'));
    }

    navigateToSignin() {
        this._router.navigate(['/auth/signin']);
    }

    signin(f: NgForm): void {
        this.sendSigninRequest(f.value.init_token);
    }

    sendSigninRequest(initToken?: string): void {
        this.loadingSignin = true;
        this._cd.markForCheck();
        this._authenticationService.signin(this.consumerType, this.code, this.state, initToken)
            .pipe(finalize(() => {
                this.loading = false;
                this.loadingSignin = false;
                this._cd.markForCheck();
            }))
            .subscribe(_ => {
                this._router.navigate([
                    (this.payloadData && this.payloadData.redirect_uri) ? this.payloadData.redirect_uri : '/home'
                ]);
            }, () => {
                this.showErrorMessage = true;
            });
    }
}
