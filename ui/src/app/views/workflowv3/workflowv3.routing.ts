import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { ProjectForWorkflowResolver, ProjectResolver } from 'app/service/services.module';
import { WorkflowV3RunComponent } from './run/workflowv3-run.component';
import { WorkflowV3ShowComponent } from './show/workflowv3-show.component';
import { WorkflowV3Component } from './workflowv3.component';
import { WorkflowV3Module } from './workflowv3.module';

const workflowRoutes: Routes = [
    {
        path: ':workflowName',
        component: WorkflowV3Component,
        data: {
            title: '{workflowName} • Workflow V3'
        },
        resolve: {
            project: ProjectForWorkflowResolver
        },
        children: [
            {
                path: '', component: WorkflowV3ShowComponent,
            },
            {
                path: 'run/:number', component: WorkflowV3RunComponent,
                resolve: {
                    project: ProjectResolver
                },
                data: {
                    title: '#{number} • {workflowName} • Workflow V3'
                },
            }
        ]
    }
];


export const workflowV3Routing: ModuleWithProviders<WorkflowV3Module> = RouterModule.forChild(workflowRoutes);
