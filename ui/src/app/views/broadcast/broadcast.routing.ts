import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {BroadcastDetailsComponent} from './details/broadcast.details.component';
import {BroadcastListComponent} from './list/broadcast.list.component';

const routes: Routes = [
    {
        path: '',
        component: BroadcastListComponent,
        canActivate: [CanActivateAuthRoute],
    },
    {
        path: ':id',
        component: BroadcastDetailsComponent,
        canActivate: [CanActivateAuthRoute],
    }
];

export const broadcastRouting: ModuleWithProviders = RouterModule.forChild(routes);
