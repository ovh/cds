import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { ProjectV2Module } from 'app/views/projectv2/project.module';
import { ProjectV2ShowComponent } from 'app/views/projectv2/project.component';
import { FeatureGuard } from 'app/guard/feature.guard';
import { FeatureNames } from 'app/service/feature/feature.service';
import { AuthenticationGuard } from 'app/guard/authentication.guard';


const projectRoutes: Routes = [
    {
        path: ':key', component: ProjectV2ShowComponent, data: { title: '{key} • Project', feature: FeatureNames.AllAsCode },
        canActivate: [AuthenticationGuard, FeatureGuard],
        canActivateChild: [AuthenticationGuard],
    }
];

export const projectV2Routing: ModuleWithProviders<ProjectV2Module> = RouterModule.forChild(projectRoutes);
