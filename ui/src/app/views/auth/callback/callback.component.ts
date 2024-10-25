import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, DefaultUrlSerializer, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import {ConfigService, LinkService, UserService} from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { jws } from 'jsrsasign';
import { Subscription } from 'rxjs';
import {finalize, first} from 'rxjs/operators';
import {Store} from "@ngxs/store";

@Component({
    selector: 'app-auth-callback',
    templateUrl: './callback.html',
    styleUrls: ['./callback.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class CallbackComponent implements OnInit, OnDestroy {
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
    cmd: string;

    constructor(
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _authenticationService: AuthenticationService,
        private _configService: ConfigService,
        private _userService: UserService,
        private _linkService: LinkService,
        private _store: Store
    ) {
        this.loading = true;
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.paramsSub = this._route.params.subscribe(params => {
            this.consumerType = params['consumerType'];

            let queryParams = this._route.snapshot.queryParams;
            if (this._route.snapshot.fragment) {
                const dus = new DefaultUrlSerializer();
                const parsed = dus.parse(this._route.snapshot.fragment);
                queryParams = parsed.queryParams;
            }

            this.code = queryParams.code || queryParams.token;
            this.state = queryParams.state || queryParams.request;

            if (!this.code || !this.state) {
                this.loading = false;
                this.missingParams = true;
                this._cd.markForCheck();
                return;
            }

            // Parse JWS state
            let payload = jws.JWS.parse(this.state).payloadObj;
            this.payloadData = payload.data;

            // If the origin is cdsctl, show the code and the state for copy
            if (this.payloadData && this.payloadData.origin === 'cdsctl') {
                this._configService.getConfig().subscribe(config => {
                    this.cmd = `cdsctl login verify ${config['url.api']} ${this.consumerType} ${this.state}:${this.code}`;
                    this.loading = false;
                    this.showCTL = true;
                    this._cd.markForCheck();
                });
                return;
            }

            // If the first connection flag is set, show init token form
            if (this.payloadData && this.payloadData.is_first_connection) {
                this.loading = false;
                this.showInitTokenForm = true;
                this._cd.markForCheck();
                return;
            }

            if (this.payloadData && this.payloadData.link_user) {
                this.linkUser();
            } else {
                this.sendSigninRequest();
            }
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

    linkUser(): void {
        this._authenticationService.getMe()
            .pipe(first())
            .subscribe(u => {
                if (u && u?.user?.username) {
                    this.loadingSignin = true;
                    this._cd.markForCheck();
                    this._linkService.link(this.consumerType, this.code, this.state)
                        .pipe(first())
                        .subscribe( () => {
                            this._router.navigate(['/', 'settings', 'user', u.user.username])
                        });
                }
            });
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
                if (this.payloadData && this.payloadData.redirect_uri &&
                    !(this.payloadData.redirect_uri as string).startsWith('/auth/callback/')) {
                    let dus = new DefaultUrlSerializer();
                    this._router.navigateByUrl(dus.parse(this.payloadData.redirect_uri));
                } else {
                    this._router.navigate(['/']);
                }
            }, () => {
                this.showErrorMessage = true;
            });
    }
}
