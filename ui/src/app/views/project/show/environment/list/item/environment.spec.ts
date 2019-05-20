/* tslint:disable:no-unused-variable */

import { HttpClientTestingModule } from '@angular/common/http/testing';
import { getTestBed, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ToasterService } from 'angular2-toaster';
import { AddEnvironmentVariableInProject, DeleteEnvironmentInProject, DeleteEnvironmentVariableInProject, UpdateEnvironmentInProject, UpdateEnvironmentVariableInProject } from 'app/store/project.action';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import { Environment } from '../../../../../../model/environment.model';
import { Project } from '../../../../../../model/project.model';
import { Variable } from '../../../../../../model/variable.model';
import { ApplicationAuditService } from '../../../../../../service/application/application.audit.service';
import { AuthentificationStore } from '../../../../../../service/auth/authentification.store';
import { EnvironmentAuditService } from '../../../../../../service/environment/environment.audit.service';
import { EnvironmentService } from '../../../../../../service/environment/environment.service';
import { NavbarService } from '../../../../../../service/navbar/navbar.service';
import { PipelineService } from '../../../../../../service/pipeline/pipeline.service';
import { ProjectAuditService } from '../../../../../../service/project/project.audit.service';
import { ProjectService } from '../../../../../../service/project/project.service';
import { ProjectStore } from '../../../../../../service/project/project.store';
import { ServicesModule, WorkflowRunService } from '../../../../../../service/services.module';
import { VariableService } from '../../../../../../service/variable/variable.service';
import { SharedModule } from '../../../../../../shared/shared.module';
import { ToastService } from '../../../../../../shared/toast/ToastService';
import { VariableEvent } from '../../../../../../shared/variable/variable.event.model';
import { ProjectModule } from '../../../../project.module';
import { ProjectEnvironmentComponent } from './environment.component';
import {WorkflowService} from 'app/service/workflow/workflow.service';
describe('CDS: Environment Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ProjectAuditService,
                EnvironmentAuditService,
                ApplicationAuditService,
                ProjectStore,
                ProjectService,
                TranslateService,
                NavbarService,
                { provide: ToastService, useClass: MockToast },
                TranslateLoader,
                TranslateParser,
                ToasterService,
                VariableService,
                EnvironmentService,
                PipelineService,
                AuthentificationStore,
                WorkflowService,
                WorkflowRunService
            ],
            imports: [
                ProjectModule,
                NgxsStoreModule,
                SharedModule,
                ServicesModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

        this.injector = getTestBed();
    });

    afterEach(() => {
        this.injector = undefined;
    });

    it('should rename environment', () => {
        // Create component
        let fixture = TestBed.createComponent(ProjectEnvironmentComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project = new Project();
        project.key = 'key1';

        let envs = new Array<Environment>();
        let e = new Environment();
        e.name = 'prod';
        e.permission = 7;
        envs.push(e);
        project.environments = envs;

        fixture.componentInstance.project = project;
        fixture.componentInstance.environment = e;

        fixture.detectChanges(true);

        let compiled = fixture.debugElement.nativeElement;

        fixture.detectChanges(true);
        fixture.componentInstance.environment.name = 'production';
        e.name = 'production';
        let store: Store = this.injector.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => {
            return of(null);
        });

        fixture.componentInstance.renameEnvironment();
        expect(store.dispatch).toHaveBeenCalledWith(new UpdateEnvironmentInProject({
            projectKey: 'key1',
            environmentName: 'prod',
            changes: e
        }));
    });

    it('should delete environment', () => {
        // Create component
        let fixture = TestBed.createComponent(ProjectEnvironmentComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project = new Project();
        project.key = 'key1';

        let envs = new Array<Environment>();
        let e = new Environment();
        e.name = 'prod';
        e.permission = 7;
        envs.push(e);
        project.environments = envs;

        fixture.componentInstance.project = project;
        fixture.componentInstance.environment = e;

        fixture.detectChanges(true);


        let store: Store = this.injector.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => {
            return of(null);
        });

        let compiled = fixture.debugElement.nativeElement;
        // Delete poller
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges(true);

        compiled.querySelector('.ui.red.button.active').click();

        expect(store.dispatch).toHaveBeenCalledWith(
            new DeleteEnvironmentInProject({
                projectKey: 'key1',
                environment: e
            })
        );
    });

    it('should add/update/delete an environment variable', () => {
        // Create component
        let fixture = TestBed.createComponent(ProjectEnvironmentComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project = new Project();
        project.key = 'key1';

        let envs = new Array<Environment>();
        let e = new Environment();
        e.name = 'prod';
        envs.push(e);
        project.environments = envs;

        fixture.componentInstance.project = project;
        fixture.componentInstance.environment = e;

        let v: Variable = new Variable();
        v.name = 'foo';
        v.value = 'bar';
        v.type = 'string';
        let event: VariableEvent = new VariableEvent('add', v);

        // Add variable

        let store: Store = this.injector.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => {
            return of(null);
        });

        fixture.componentInstance.variableEvent(event);
        expect(store.dispatch).toHaveBeenCalledWith(new AddEnvironmentVariableInProject({
            projectKey: 'key1',
            environmentName: 'prod',
            variable: v
        }));

        // Update variable
        event.type = 'update';
        fixture.componentInstance.variableEvent(event);
        expect(store.dispatch).toHaveBeenCalledWith(new UpdateEnvironmentVariableInProject({
            projectKey: 'key1',
            environmentName: 'prod',
            variableName: v.name,
            changes: v
        }));

        // Delete variable
        event.type = 'delete';
        fixture.componentInstance.variableEvent(event);
        expect(store.dispatch).toHaveBeenCalledWith(new DeleteEnvironmentVariableInProject({
            projectKey: 'key1',
            environmentName: 'prod',
            variable: v
        }));
    });
});

class MockToast {
    success(title: string, msg: string) {

    }
}
