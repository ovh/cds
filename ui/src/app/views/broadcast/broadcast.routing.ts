import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { BroadcastModule } from 'app/views/broadcast/broadcast.module';
import { BroadcastDetailsComponent } from './details/broadcast.details.component';
import { BroadcastListComponent } from './list/broadcast.list.component';

const routes: Routes = [
    {
        path: '',
        component: BroadcastListComponent,
        canActivate: [AuthenticationGuard],
    },
    {
        path: ':id',
        component: BroadcastDetailsComponent,
        canActivate: [AuthenticationGuard],
    }
];

export const broadcastRouting: ModuleWithProviders<BroadcastModule> = RouterModule.forChild(routes);
