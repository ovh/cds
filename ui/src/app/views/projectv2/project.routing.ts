import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { ProjectV2Module } from 'app/views/projectv2/project.module';
import { ProjectV2ShowComponent } from 'app/views/projectv2/project.component';


const projectRoutes: Routes = [
    {
        path: ':key', component: ProjectV2ShowComponent, data: { title: '{key} • Project' },
        canActivate: [AuthenticationGuard],
        canActivateChild: [AuthenticationGuard],
    }
];

export const projectV2Routing: ModuleWithProviders<ProjectV2Module> = RouterModule.forChild(projectRoutes);
