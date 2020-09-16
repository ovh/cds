import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { WorkflowModule } from 'app/views/workflow/workflow.module';
import { ProjectForWorkflowResolver, ProjectResolver } from '../../service/project/project.resolver';
import { WorkflowAddComponent } from './add/workflow.add.component';
import { WorkflowNodeRunComponent } from './run/node/workflow.run.node.component';
import { WorkflowRunComponent } from './run/workflow.run.component';
import { WorkflowShowComponent } from './show/workflow.component';
import { WorkflowComponent } from './workflow.component';

const workflowRoutes: Routes = [
    {
        path: '',
        component: WorkflowAddComponent,
        canActivate: [AuthenticationGuard],
        canActivateChild: [AuthenticationGuard],
        resolve: {
            project: ProjectForWorkflowResolver
        },
        data: {
            title: 'Add • Workflow'
        },
    },
    {
        path: ':workflowName',
        component: WorkflowComponent,
        canActivate: [AuthenticationGuard],
        canActivateChild: [AuthenticationGuard],
        data: {
            title: '{workflowName} • Workflow'
        },
        resolve: {
            project: ProjectForWorkflowResolver
        },
        children: [
            {
                path: '', component: WorkflowShowComponent,
            },
            {
                path: 'run/:number', component: WorkflowRunComponent,
                resolve: {
                    project: ProjectResolver
                },
                data: {
                    title: '#{number} • {workflowName}'
                },
            },
            {
                path: 'run/:number/node/:nodeId', component: WorkflowNodeRunComponent,
                resolve: {
                    project: ProjectForWorkflowResolver
                },
                data: {
                    title: 'Pipeline {name} • #{number} • {workflowName}'
                },
            }
        ]
    }
];


export const workflowRouting: ModuleWithProviders<WorkflowModule> = RouterModule.forChild(workflowRoutes);
