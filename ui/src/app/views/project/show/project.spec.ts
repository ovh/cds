/* tslint:disable:no-unused-variable */
import {TestBed, getTestBed, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {ToasterService} from 'angular2-toaster/angular2-toaster';
import {TranslateParser} from '@ngx-translate/core';
import {ProjectStore} from '../../../service/project/project.store';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';
import {ProjectService} from '../../../service/project/project.service';
import {PipelineService} from '../../../service/pipeline/pipeline.service';
import {VariableService} from '../../../service/variable/variable.service';
import {EnvironmentService} from '../../../service/environment/environment.service';
import {ToastService} from '../../../shared/toast/ToastService';
import {ProjectModule} from '../project.module';
import {SharedModule} from '../../../shared/shared.module';
import {ServicesModule} from '../../../service/services.module';
import {ProjectShowComponent} from './project.component';
import {ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Observable';
import {Project} from '../../../model/project.model';
import {Map} from 'immutable';
import {GroupPermission} from '../../../model/group.model';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {User} from '../../../model/user.model';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import 'rxjs/add/observable/of';
describe('CDS: Project Show Component', () => {

    let injector: Injector;
    let backend: MockBackend;
    let authStore: AuthentificationStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                AuthentificationStore,
                MockBackend,
                {provide: XHRBackend, useClass: MockBackend},
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                PipelineService,
                VariableService,
                EnvironmentService,
                ToasterService,
                TranslateService,
                TranslateParser,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                {provide: ToastService, useClass: MockToast}
            ],
            imports: [
                ProjectModule,
                SharedModule,
                ServicesModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);
        authStore = injector.get(AuthentificationStore);
        authStore.addUser(new User(), false);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        authStore = undefined;
    });

    it('it should add/update/delete group', fakeAsync(() => {
        let projectStore: ProjectStore = injector.get(ProjectStore);

        let p: Project = new Project();
        p.key = 'key1';
        spyOn(projectStore, 'getProjects').and.callFake(() => {
            let mapProject: Map<string, Project> = Map<string, Project>();
            return Observable.of(mapProject.set('key1', p));
        });

        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 7;
        let event: PermissionEvent = new PermissionEvent('add', gp);

        // Add variable
        spyOn(projectStore, 'addProjectPermission').and.callFake(() => {
            return Observable.of(p);
        });
        fixture.componentInstance.groupEvent(event);
        expect(projectStore.addProjectPermission).toHaveBeenCalledWith('key1', gp);

        // Update variable
        event.type = 'update';
        spyOn(projectStore, 'updateProjectPermission').and.callFake(() => {
            return Observable.of(p);
        });
        fixture.componentInstance.groupEvent(event);
        expect(projectStore.updateProjectPermission).toHaveBeenCalledWith('key1', gp);

        // Delete variable
        event.type = 'delete';
        spyOn(projectStore, 'removeProjectPermission').and.callFake(() => {
            return Observable.of(p);
        });
        fixture.componentInstance.groupEvent(event);
        expect(projectStore.removeProjectPermission).toHaveBeenCalledWith('key1', gp);
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({key: 'key1'});

        this.queryParams = Observable.of({tab: 'application'});
        this.snapshot = new MockActivatedRouteSnapshot();
    }
}

class MockActivatedRouteSnapshot extends ActivatedRouteSnapshot {

}
