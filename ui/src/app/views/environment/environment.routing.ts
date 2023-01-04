import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { ProjectResolver } from 'app/service/project/project.resolver';
import { EnvironmentModule } from 'app/views/environment/environment.module';
import { EnvironmentAddComponent } from './add/environment.add.component';
import { EnvironmentShowComponent } from './show/environment.show.component';

const environmentRoutes: Routes = [
    {
        path: '',
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
