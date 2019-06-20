import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { CanActivateAuthRoute } from '../../service/auth/authenRouteActivate';
import { ProjectAddComponent } from './add/project.add.component';
import { ProjectListComponent } from './list/project.list.component';
import { ProjectShowComponent } from './show/project.component';

const projectRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: ProjectAddComponent, data: { title: 'Add • Project' } },
            { path: 'list/all', component: ProjectListComponent, data: { title: 'List • Project' } },
            { path: ':key', component: ProjectShowComponent, data: { title: '{key} • Project' } },
            {
                path: ':key/workflow', loadChildren:
                    () => import('app/views/workflow/workflow.module').then(m => m.WorkflowModule)
            },
            {
                path: ':key/environment', loadChildren:
                    () => import('app/views/environment/environment.module').then(m => m.EnvironmentModule)
            },
            {
                path: ':key/application', loadChildren:
                    () => import('app/views/application/application.module').then(m => m.ApplicationModule)
            },
            {
                path: ':key/pipeline', loadChildren:
                    () => import('app/views/pipeline/pipeline.module').then(m => m.PipelineModule)
            }
        ]
    }
];

export const projectRouting: ModuleWithProviders = RouterModule.forChild(projectRoutes);
