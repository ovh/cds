import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {WarningShowComponent} from './show/warning.show.component';

const warningRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            {
                path: 'show', component: WarningShowComponent
            }
        ]
    }
];


export const warningRouting: ModuleWithProviders = RouterModule.forChild(warningRoutes);
