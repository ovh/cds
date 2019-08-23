import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AuthenticationService } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-auth-ask-signin',
    templateUrl: './ask-signin.html',
    styleUrls: ['./ask-signin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})

@AutoUnsubscribe()
export class AskSigninComponent implements OnInit {
    paramsSub: Subscription;
    showCTL: boolean;
    code: string;
    state: string;
    loading: boolean;
    showErrorMessage: boolean;

    constructor(
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef,
        private _authenticationService: AuthenticationService
    ) {
        this.loading = true;
    }

    ngOnInit() {
        this.paramsSub = this._route.params.subscribe(params => {
            let consumerType = params['consumerType'];
            let redirectURI = this._route.snapshot.queryParamMap.get('redirect_uri');
            let requireMFA = this._route.snapshot.queryParamMap.get('require_mfa') === 'true';

            this._authenticationService.askSignin(consumerType, redirectURI, requireMFA)
                .pipe(finalize(() => {
                    this.loading = false;
                    this._cd.detectChanges();
                }))
                .subscribe(redirect => {
                    if (redirect.method.toLowerCase() === ('get')) {
                        window.location.replace(redirect.url);
                    } else {
                        let form = document.createElement('form');
                        form.method = redirect.method;
                        form.action = redirect.url;
                        form.enctype = redirect.content_type;

                        Object.keys(redirect.body).forEach((k) => {
                            let input = document.createElement('input');
                            input.type = 'hidden';
                            input.name = k;
                            input.value = redirect.body[k];
                            form.append(input);
                        });

                        document.body.append(form);
                        form.submit();
                    }
                }, () => {
                    this.showErrorMessage = true;
                });
        });
    }
}
