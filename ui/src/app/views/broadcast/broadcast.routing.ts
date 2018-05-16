import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {BroadcastListComponent} from './list/broadcast.list.component';
import {BroadcastDetailsComponent} from './details/broadcast.details.component';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';

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
