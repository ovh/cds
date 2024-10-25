import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { PipelineModule } from 'app/views/pipeline/pipeline.module';
import { ApplicationQueryParamResolver } from '../../service/application/application.resolver';
import { ProjectResolver } from '../../service/project/project.resolver';
import { PipelineAddComponent } from './add/pipeline.add.component';
import { PipelineShowComponent } from './show/pipeline.show.component';

const pipelineRoutes: Routes = [
    {
        path: '',
        children: [
            {
                path: '',
                component: PipelineAddComponent,
                resolve: {
                    project: ProjectResolver
                },
                data: { title: 'Add • Pipeline' }
            },
            {
                path: ':pipName',
                component: PipelineShowComponent,
                data: { title: '{pipName} • Pipeline' },
                resolve: {
                    project: ProjectResolver,
                    application: ApplicationQueryParamResolver
                }
            }
        ]
    }
];

export const pipelineRouting: ModuleWithProviders<PipelineModule> = RouterModule.forChild(pipelineRoutes);
