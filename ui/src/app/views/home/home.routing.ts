import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { HomeModule } from 'app/views/home/home.module';
import { HomeComponent } from './home.component';

const routes: Routes = [
    {
        path: '',
        component: HomeComponent,
        canActivate: [AuthenticationGuard]
    }
];

export const homeRouting: ModuleWithProviders<HomeModule> = RouterModule.forChild(routes);
