import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { CanActivateAuthRoute } from 'app/service/auth/authenRouteActivate';
import { ProjectResolver } from 'app/service/project/project.resolver';
import { EnvironmentAddComponent } from './add/environment.add.component';
import { EnvironmentShowComponent } from './show/environment.show.component';

const environmentRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            {
                path: '', component: EnvironmentAddComponent,
                data: { title: 'Add • Environment' },
                resolve: {
                    project: ProjectResolver
                }
            },
            {
                path: ':envName',
                component: EnvironmentShowComponent,
                data: { title: '{envName} • Environment' },
                resolve: {
                    project: ProjectResolver
                }
            }
        ]
    }
];


export const environmentRouting: ModuleWithProviders = RouterModule.forChild(environmentRoutes);
