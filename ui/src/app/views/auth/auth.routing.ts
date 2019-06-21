import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
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
                data: { title: 'CDS • Sign in' }
            },
            {
                path: 'ask-reset',
                component: ResetComponent,
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
        ]
    }
];

export const authRouting: ModuleWithProviders = RouterModule.forChild(routes);
