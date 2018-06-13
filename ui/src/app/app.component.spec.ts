/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed} from '@angular/core/testing';
import {AppModule} from './app.module';
import {AppComponent} from './app.component';
import {Injector} from '@angular/core';
import {AuthentificationStore} from './service/auth/authentification.store';
import {User} from './model/user.model';
import {SharedModule} from './shared/shared.module';
import {ProjectStore} from './service/project/project.store';
import {ApplicationStore} from './service/application/application.store';
import {ProjectService} from './service/project/project.service';
import {ActivatedRoute} from '@angular/router';
import {APP_BASE_HREF} from '@angular/common';
import {Observable} from 'rxjs/Observable';
import {ApplicationService} from './service/application/application.service';
import {PipelineService} from './service/pipeline/pipeline.service';
import {PipelineStore} from './service/pipeline/pipeline.store';
import {LastModification} from './model/lastupdate.model';
import {AppService} from './app.service';
import {RouterTestingModule} from '@angular/router/testing';
import {Pipeline} from './model/pipeline.model';
import {Application} from './model/application.model';
import {Project} from './model/project.model';
import {first} from 'rxjs/operators';
import 'rxjs/add/observable/of';

describe('App: CDS', () => {

    let injector: Injector;
    let authStore: AuthentificationStore;
    let projectStore: ProjectStore;
    let applicationStore: ApplicationStore;
    let pipelineStore: PipelineStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                AuthentificationStore,
                { provide: APP_BASE_HREF, useValue: '/' },
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: ProjectService, useClass: MockProjectService },
                { provide: ApplicationService, useClass: MockApplicationService},
                { provide: PipelineService, useClass: MockPipelineService},
                { provide: ActivatedRoute, useClass: MockActivatedRoutes}
            ],
            imports : [
                AppModule,
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

        injector = getTestBed();
        authStore = injector.get(AuthentificationStore);
        projectStore = injector.get(ProjectStore);
        applicationStore = injector.get(ApplicationStore);
        pipelineStore = injector.get(PipelineStore);
    });

    afterEach(() => {
        injector = undefined;
        authStore = undefined;
        projectStore = undefined;
        applicationStore = undefined;
        pipelineStore = undefined;
    });


    it('should create the app', () => {
        let fixture = TestBed.createComponent(AppComponent);
        let app = fixture.debugElement.componentInstance;
        expect(app).toBeTruthy();
    });


    it('should render a navbar', () => {
        let fixture = TestBed.createComponent(AppComponent);
        let compiled = fixture.debugElement.nativeElement;

        expect(fixture.componentInstance.isConnected).toBeFalsy('IsConnected flag must be false');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must not have the css class "connected"');

        fixture.componentInstance.ngOnInit();
        authStore.addUser(new User(), false);

        expect(fixture.componentInstance.isConnected).toBeTruthy('IsConnected flag must be true');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must have connected css class');
    });
});
class MockActivatedRoutes {

    snapshot: {
        params: {}
    };

    constructor() {
        this.snapshot = {
            params: {
                'key': 'key1',
                'appName': 'app2',
                'pipName': 'pip2'
            }
        };
    }
}

class MockProjectService extends ProjectService {
    callKEY1 = 0;
    getProject(key: string) {
        switch (key) {
            case 'key1':
                if (this.callKEY1 === 0) {
                    this.callKEY1++;
                    let proj = new Project();
                    proj.key = 'key1';
                    proj.name = 'project1';
                    proj.last_modified = '2017-05-11T10:20:22.874779+02:00';
                    return Observable.of(proj);
                } else {
                    let proj = new Project();
                    proj.key = 'key1';
                    proj.name = 'project1';
                    proj.last_modified = '2017-06-11T10:20:22.874779+02:00';
                    return Observable.of(proj);
                }
            case 'key2':
                let proj2 = new Project();
                proj2.key = 'key2';
                proj2.name = 'project2';
                proj2.last_modified = '2017-05-11T10:20:22.874779+02:00';
                return Observable.of(proj2);
        }

    }
}

class MockApplicationService extends ApplicationService {
    callAPP2 = 0;

    getApplication(key: string, appName: string, filter?: {branch: string, remote: string}) {
        if (key === 'key1') {
            if (appName === 'app1') {
                let app = new Application();
                app.name = 'app1';
                app.last_modified = '2017-05-11T10:20:22.874779+02:00';
                return Observable.of(app);
            }
            if (appName === 'app2') {
                if (this.callAPP2 === 0) {
                    this.callAPP2++;
                    let app = new Application();
                    app.name = 'app2';
                    app.last_modified = '2017-05-11T10:20:22.874779+02:00';
                    return Observable.of(app);
                } else {
                    let app = new Application();
                    app.name = 'app2';
                    app.last_modified = '2017-06-11T10:20:22.874779+02:00';
                    return Observable.of(app);
                }

            }
        }
        if (key === 'key2') {
            if (appName === 'app3') {
                let app = new Application();
                app.name = 'app3';
                app.last_modified = '2017-05-11T10:20:22.874779+02:00';
                return Observable.of(app);
            }
        }
    }
}

class MockPipelineService extends PipelineService {
    callPIP2 = 0;
    getPipeline(key: string, pipName: string) {
        if (key === 'key1') {
            if (pipName === 'pip1') {
                let pip = new Pipeline();
                pip.name = 'pip1';
                pip.last_modified = 1494490822;
                return Observable.of(pip);
            }
            if (pipName === 'pip2') {
                if (this.callPIP2 === 0) {
                    this.callPIP2++;
                    let pip = new Pipeline();
                    pip.name = 'pip1';
                    pip.last_modified = 1494490822;
                    return Observable.of(pip);
                } else {
                    let pip = new Pipeline();
                    pip.name = 'pip1';
                    pip.last_modified = 1497169222;
                    return Observable.of(pip);
                }

            }
        }
        if (key === 'key2') {
            if (pipName === 'pip3') {
                let pip = new Pipeline();
                pip.name = 'pip3';
                pip.last_modified = 1494490822;
                return Observable.of(pip);
            }
        }
    }
}
