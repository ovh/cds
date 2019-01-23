import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { CanActivateAuthRoute } from '../../service/auth/authenRouteActivate';
import { ProjectForApplicationResolver, ProjectForWorkflowResolver } from '../../service/project/project.resolver';
import { ApplicationAddComponent } from './add/application.add.component';
import { ApplicationShowComponent } from './show/application.component';

const applicationRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: ApplicationAddComponent ,
                data: { title: 'Add • Application' },
                resolve: {
                    project: ProjectForWorkflowResolver
                }
            },
            { path: ':appName',
                component: ApplicationShowComponent,
                data: { title: '{appName} • Application' },
                resolve: {
                    project: ProjectForApplicationResolver
                }
            }
        ]
    }
];


export const applicationRouting: ModuleWithProviders = RouterModule.forChild(applicationRoutes);
