import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { NoAuthenticationGuard } from 'app/guard/no-authentication.guard';
import { AuthModule } from 'app/views/auth/auth.module';
import { AskResetComponent } from './ask-reset/ask-reset';
import { AskSigninComponent } from './ask-signin/ask-signin.component';
import { AuthComponent } from './auth.component';
import { CallbackComponent } from './callback/callback.component';
import { ResetComponent } from './reset/reset';
import { SigninComponent } from './signin/signin';
import { VerifyComponent } from './verify/verify.component';

const routes: Routes = [
    {
        path: '',
        component: AuthComponent,
        children: [
            { path: '', redirectTo: 'signin', pathMatch: 'full' },
            {
                path: 'signin',
                component: SigninComponent,
                data: { title: 'CDS • Sign in' },
                canActivate: [NoAuthenticationGuard]
            },
            {
                path: 'ask-reset',
                component: AskResetComponent,
                data: { title: 'CDS • Ask password reset' }
            },
            {
                path: 'reset',
                component: ResetComponent,
                data: { title: 'CDS • Reset password' }
            },
            {
                path: 'verify',
                component: VerifyComponent,
                data: { title: 'CDS • Verify account' }
            },
            {
                path: 'callback/:consumerType',
                component: CallbackComponent,
                data: { title: 'CDS • Authentication callback' }
            },
            {
                path: 'ask-signin/:consumerType',
                component: AskSigninComponent,
                data: { title: 'CDS • Authentication' }
            },
        ]
    }
];

export const authRouting: ModuleWithProviders<AuthModule> = RouterModule.forChild(routes);
