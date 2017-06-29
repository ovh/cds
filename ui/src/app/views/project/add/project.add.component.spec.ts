/* tslint:disable:no-unused-variable */
import {TestBed, getTestBed, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader} from 'ng2-translate/ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {ToasterService} from 'angular2-toaster/angular2-toaster';
import {TranslateParser} from 'ng2-translate';
import {ProjectStore} from '../../../service/project/project.store';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';
import {ProjectService} from '../../../service/project/project.service';
import {ToastService} from '../../../shared/toast/ToastService';
import {ProjectModule} from '../project.module';
import {SharedModule} from '../../../shared/shared.module';
import {Observable} from 'rxjs/Rx';
import {ProjectAddComponent} from './project.add.component';
import {GroupService} from '../../../service/group/group.service';
import {GroupPermission, Group} from '../../../model/group.model';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {Router} from '@angular/router';
import {Project} from '../../../model/project.model';

describe('CDS: Project Show Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                MockBackend,
                {provide: XHRBackend, useClass: MockBackend},
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                ToasterService,
                TranslateService,
                TranslateParser,
                GroupService,
                {provide: ToastService, useClass: MockToast}
            ],
            imports: [
                ProjectModule,
                SharedModule,
                RouterTestingModule.withRoutes([]),

            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);

    });

    afterEach(() => {
        injector = undefined;
    });


    it('it should create a project', fakeAsync(() => {
        let projectStore: ProjectStore = injector.get(ProjectStore);
        let router: Router = injector.get(Router);

        spyOn(projectStore, 'createProject').and.callFake(() => {
            return Observable.of(true);
        });

        spyOn(router, 'navigate').and.callFake(() => {
            return;
        });

        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectAddComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.project.name = 'FooProject';
        fixture.componentInstance.project.key = 'BAR';

        fixture.componentInstance.project.groups = new Array<GroupPermission>();
        fixture.componentInstance.group = new Group();
        fixture.componentInstance.group.name = 'foo';

        fixture.componentInstance.createProject();

        let project = new Project();
        project.name = 'FooProject';
        project.key = 'BAR';
        project.groups = new Array<GroupPermission>();
        project.groups.push(new GroupPermission());
        project.groups[0].group = new Group();
        project.groups[0].group.name = 'foo';
        project.groups[0].permission = 7;
        expect(projectStore.createProject).toHaveBeenCalledWith(project);
        expect(router.navigate).toHaveBeenCalled();
    }));

    it('it should generate an project key', fakeAsync(() => {
        let fixture = TestBed.createComponent(ProjectAddComponent);
        fixture.componentInstance.generateKey('^r%t*$f#|m');
        expect(fixture.componentInstance.project.key).toBe('RTFM');

    }));

    it('it should generate errors', fakeAsync(() => {
        let fixture = TestBed.createComponent(ProjectAddComponent);
        fixture.componentInstance.addSshKey = true;
        fixture.componentInstance.createProject();

        expect(fixture.componentInstance.nameError).toBeTruthy();
        expect(fixture.componentInstance.keyError).toBeTruthy();
        expect(fixture.componentInstance.groupError).toBeTruthy();
        expect(fixture.componentInstance.sshError).toBeTruthy();

        // pattern error
        fixture.componentInstance.project.key = 'aze';
        fixture.componentInstance.createProject();
        expect(fixture.componentInstance.keyError).toBeTruthy();

        // no group with write right
        fixture.componentInstance.project.groups = new Array<GroupPermission>();
        let gp = new GroupPermission();
        gp.permission = 4;
        fixture.componentInstance.project.groups.push(gp);
        fixture.componentInstance.createProject();
        expect(fixture.componentInstance.groupError).toBeTruthy();
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}
