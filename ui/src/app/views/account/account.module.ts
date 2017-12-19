import {NgModule} from '@angular/core';
import {LoginComponent} from './login/login.component';
import {accountRouting} from './account.routing';
import {PasswordComponent} from './password/password.component';
import {SignUpComponent} from './signup/signup.component';
import {VerifyComponent} from './verify/verify.component';
import {SharedModule} from '../../shared/shared.module';

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
