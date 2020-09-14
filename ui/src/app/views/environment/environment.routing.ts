import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { ProjectResolver } from 'app/service/project/project.resolver';
import { EnvironmentModule } from 'app/views/environment/environment.module';
import { EnvironmentAddComponent } from './add/environment.add.component';
import { EnvironmentShowComponent } from './show/environment.show.component';

const environmentRoutes: Routes = [
    {
        path: '',
        canActivate: [AuthenticationGuard],
        canActivateChild: [AuthenticationGuard],
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

export const environmentRouting: ModuleWithProviders<EnvironmentModule> = RouterModule.forChild(environmentRoutes);
