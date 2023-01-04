import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { HomeModule } from 'app/views/home/home.module';
import { HomeComponent } from './home.component';

const routes: Routes = [
    {
        path: '',
        component: HomeComponent
    }
];

export const homeRouting: ModuleWithProviders<HomeModule> = RouterModule.forChild(routes);
