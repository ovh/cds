import {NgModule} from '@angular/core';
import {SharedModule} from 'app/shared/shared.module';
import {accountRouting} from './account.routing';
import {LoginComponent} from './login/login.component';
import {PasswordComponent} from './password/password.component';
import {SignUpComponent} from './signup/signup.component';
import {VerifyComponent} from './verify/verify.component';

@NgModule({
    declarations: [
        LoginComponent,
        PasswordComponent,
        SignUpComponent,
        VerifyComponent,
    ],
    imports: [
        SharedModule,
        accountRouting
    ]
})
export class AccountModule {
}
