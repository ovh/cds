import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { ProjectV2Module } from 'app/views/projectv2/project.module';
import { ProjectV2ShowComponent } from 'app/views/projectv2/project.component';
import { FeatureGuard } from 'app/guard/feature.guard';
import { FeatureNames } from 'app/service/feature/feature.service';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { ProjectV2RepositoryAddComponent } from 'app/views/projectv2/vcs/repository/project.repository.add.component';
import { Projectv2Resolver } from 'app/service/project/project.resolver';
import {
    ProjectV2RepositoryShowComponent
} from 'app/views/projectv2/vcs/repository/show/project.repository.show.component';
import {ProjectV2WorkerModelShowComponent} from "./vcs/repository/workermodel/show/project.workermodel.show.component";


const projectRoutes: Routes = [
    {
        path: ':key', component: ProjectV2ShowComponent, data: { title: '{key} • Project', feature: FeatureNames.AllAsCode },
        canActivate: [AuthenticationGuard, FeatureGuard],
        canActivateChild: [AuthenticationGuard],
        children: [
            {
                path: 'vcs/:vcsName/repository',
                component: ProjectV2RepositoryAddComponent,
                data: { title: 'Add • Repository' },
                resolve: {
                    project: Projectv2Resolver,
                },
            },
            {
                path: 'vcs/:vcsName/repository/:repoName',
                component: ProjectV2RepositoryShowComponent,
                data: { title: '{repoName} • Repository' },
                resolve: {
                    project: Projectv2Resolver,
                },
            },
            {
                path: 'vcs/:vcsName/repository/:repoName/workermodel/:workerModelName',
                component: ProjectV2WorkerModelShowComponent,
                data: { title: '{workerModelName} • Worker Model' },
                resolve: {
                    project: Projectv2Resolver,
                },
            }
        ]
    }
];

export const projectV2Routing: ModuleWithProviders<ProjectV2Module> = RouterModule.forChild(projectRoutes);
