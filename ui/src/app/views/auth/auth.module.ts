import { NgModule } from '@angular/core';
import { SharedModule } from '../../shared/shared.module';
import { AskResetComponent } from './ask-reset/ask-reset';
import { AskSigninComponent } from './ask-signin/ask-signin.component';
import { AuthComponent } from './auth.component';
import { authRouting } from './auth.routing';
import { CallbackComponent } from './callback/callback.component';
import { ResetComponent } from './reset/reset';
import { SigninComponent } from './signin/signin';
import { VerifyComponent } from './verify/verify.component';

@NgModule({
    declarations: [
        AuthComponent,
        SigninComponent,
        AskResetComponent,
        AskSigninComponent,
        ResetComponent,
        VerifyComponent,
        CallbackComponent
    ],
    imports: [
        SharedModule,
        authRouting
    ]
})
export class AuthModule {
}
