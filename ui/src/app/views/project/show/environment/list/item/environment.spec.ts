/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, fakeAsync, tick} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {ProjectEnvironmentComponent} from './environment.component';
import {AuthentificationStore} from '../../../../../../service/auth/authentification.store';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ProjectService} from '../../../../../../service/project/project.service';
import {EnvironmentService} from '../../../../../../service/environment/environment.service';
import {PipelineService} from '../../../../../../service/pipeline/pipeline.service';
import {ProjectModule} from '../../../../project.module';
import {SharedModule} from '../../../../../../shared/shared.module';
import {ServicesModule} from '../../../../../../service/services.module';
import {Project} from '../../../../../../model/project.model';
import {Environment} from '../../../../../../model/environment.model';
import {ToasterService} from 'angular2-toaster';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {VariableService} from '../../../../../../service/variable/variable.service';
import {Observable} from 'rxjs/Rx';
import {VariableEvent} from '../../../../../../shared/variable/variable.event.model';
import {Variable} from '../../../../../../model/variable.model';
import {ProjectAuditService} from '../../../../../../service/project/project.audit.service';
import {EnvironmentAuditService} from '../../../../../../service/environment/environment.audit.service';
import {ApplicationAuditService} from '../../../../../../service/application/application.audit.service';
import {HttpClientTestingModule} from '@angular/common/http/testing';

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
                { provide: XHRBackend, useClass: MockBackend },
                { provide: ToastService, useClass: MockToast },
                TranslateLoader,
                TranslateParser,
                ToasterService,
                VariableService,
                EnvironmentService,
                PipelineService,
                AuthentificationStore
            ],
            imports : [
                ProjectModule,
                SharedModule,
                ServicesModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

        this.injector = getTestBed();
    });

    afterEach(() => {
        this.injector = undefined;
    });

    it('should rename environment', fakeAsync( () => {
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

        fixture.detectChanges();
        tick(250);

        let compiled = fixture.debugElement.nativeElement;
        let inputName = compiled.querySelector('input[name="envname"]');
        inputName.value = 'production';
        inputName.dispatchEvent(new Event('input'));
        inputName.dispatchEvent(new Event('keydown'));

        fixture.detectChanges();
        tick(250);

        e.name = 'production';
        let projectStore: ProjectStore = this.injector.get(ProjectStore);
        spyOn(projectStore, 'renameProjectEnvironment').and.callFake(() => {
           return Observable.of(project);
        });

        compiled.querySelector('button[name="renamebtn"]').click();
        expect(projectStore.renameProjectEnvironment).toHaveBeenCalledWith('key1', 'prod', e);
    }));

    it('should delete environment', fakeAsync( () => {
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

        fixture.detectChanges();
        tick(250);


        let projectStore: ProjectStore = this.injector.get(ProjectStore);
        spyOn(projectStore, 'deleteProjectEnvironment').and.callFake(() => {
            return Observable.of(project);
        });

        let compiled = fixture.debugElement.nativeElement;
        // Delete poller
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges();
        tick(50);
        compiled.querySelector('.ui.red.button.active').click();

        expect(projectStore.deleteProjectEnvironment).toHaveBeenCalledWith('key1', e);
    }));

    it('should add/update/delete an environment variable', fakeAsync( () => {
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

        let projectStore: ProjectStore = this.injector.get(ProjectStore);
        spyOn(projectStore, 'addEnvironmentVariable').and.callFake(() => {
            return Observable.of(project);
        });

        fixture.componentInstance.variableEvent(event);
        expect(projectStore.addEnvironmentVariable).toHaveBeenCalledWith('key1', 'prod', v);

        // Update variable
        event.type = 'update';
        spyOn(projectStore, 'updateEnvironmentVariable').and.callFake(() => {
            return Observable.of(project);
        });
        fixture.componentInstance.variableEvent(event);
        expect(projectStore.updateEnvironmentVariable).toHaveBeenCalledWith('key1', 'prod', v);

        // Delete variable
        event.type = 'delete';
        spyOn(projectStore, 'removeEnvironmentVariable').and.callFake(() => {
            return Observable.of(project);
        });
        fixture.componentInstance.variableEvent(event);
        expect(projectStore.removeEnvironmentVariable).toHaveBeenCalledWith('key1', 'prod', v);

    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}
