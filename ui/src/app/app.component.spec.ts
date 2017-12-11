/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed, fakeAsync} from '@angular/core/testing';
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


    it('should create the app', async( () => {
        let fixture = TestBed.createComponent(AppComponent);
        let app = fixture.debugElement.componentInstance;
        expect(app).toBeTruthy();
    }));


    it('should render a navbar', async(() => {
        let fixture = TestBed.createComponent(AppComponent);
        let compiled = fixture.debugElement.nativeElement;

        expect(fixture.componentInstance.isConnected).toBeFalsy('IsConnected flag must be false');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must not have the css class "connected"');

        fixture.componentInstance.startLastUpdateSSE = () => {

        };

        fixture.componentInstance.ngOnInit();
        authStore.addUser(new User(), false);

        expect(fixture.componentInstance.isConnected).toBeTruthy('IsConnected flag must be true');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must have connected css class');
    }));

    it('should update cache', fakeAsync(() => {
        // Create cache
        projectStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        projectStore.getProjects('key2').pipe(first()).subscribe(() => {}).unsubscribe();

        applicationStore.getApplications('key1', 'app1').pipe(first()).subscribe(() => {}).unsubscribe();
        applicationStore.getApplications('key1', 'app2').pipe(first()).subscribe(() => {}).unsubscribe();
        applicationStore.getApplications('key2', 'app3').pipe(first()).subscribe(() => {}).unsubscribe();

        pipelineStore.getPipelines('key1', 'pip1').pipe(first()).subscribe(() => {}).unsubscribe();
        pipelineStore.getPipelines('key1', 'pip2').pipe(first()).subscribe(() => {}).unsubscribe();
        pipelineStore.getPipelines('key2', 'pip3').pipe(first()).subscribe(() => {}).unsubscribe();


        let prj1Update = new LastModification();
        prj1Update.key = 'key1';
        prj1Update.last_modified = 1497169222;
        prj1Update.username = 'fooApp';
        prj1Update.type = 'project';

        let app1Update = new LastModification();
        app1Update.key = 'key1';
        app1Update.last_modified = 1494490822;
        app1Update.name = 'app1';
        app1Update.type = 'application';


        let app2Update = new LastModification();
        app2Update.key = 'key1';
        app2Update.last_modified = 1497169222;
        app2Update.name = 'app2';
        app2Update.username = 'fooApp';
        app2Update.type = 'application';

        let pip1Update = new LastModification();
        pip1Update.key = 'key1';
        pip1Update.last_modified = 1494490822;
        pip1Update.name = 'pip1';
        pip1Update.type = 'pipeline';

        let pip2Update = new LastModification();
        pip2Update.key = 'key1';
        pip2Update.last_modified = 1497169222;
        pip2Update.name = 'pip2';
        pip2Update.username = 'fooApp';
        pip2Update.type = 'pipeline';

        let prj2Update = new LastModification();
        prj2Update.key = 'key2';
        prj2Update.last_modified = 1497169222;
        prj2Update.type = 'project';

        let app3Update = new LastModification();
        app3Update.key = 'key2';
        app3Update.last_modified = 1497169222;
        app3Update.name = 'app3';
        app3Update.username = 'bar';
        app3Update.type = 'application';

        let pip3Update = new LastModification();
        pip3Update.key = 'key2';
        pip3Update.last_modified = 1497169222;
        pip3Update.name = 'pip3';
        pip3Update.username = 'bar';
        pip3Update.type = 'pipeline';

        let AuthStore = injector.get(AuthentificationStore);
        let user = new User();
        user.username = 'fooApp';
        AuthStore.addUser(user, true);

        let appService = injector.get(AppService);

        appService.updateCache(prj1Update);
        appService.updateCache(prj2Update);

        // Check project result
        let check = false;
        projectStore.getProjects().subscribe(projs => {
            check = true;
            expect(projs.size).toBe(1, 'Must have just 1 project');
            expect(projs.get('key1')).toBeTruthy('project key1 must be here');
            expect(projs.get('key1').last_modified).toBe('2017-06-11T10:20:22.874779+02:00', 'project key1 have to be up to date');
        }).unsubscribe();
        expect(check).toBe(true);

        appService.updateCache(app1Update);
        appService.updateCache(app2Update);
        appService.updateCache(app3Update);

        // Check application result
        let checkApp = false;
        applicationStore.getApplications('key', null).subscribe(apps => {
            checkApp = true;
            expect(apps.size).toBe(2, 'Must have 2 applications in cache');
            expect(apps.get('key1-app1')).toBeTruthy('app1 must be here');
            expect(apps.get('key1-app1').last_modified).toBe('2017-05-11T10:20:22.874779+02:00', 'No change on app1');
            expect(apps.get('key1-app2')).toBeTruthy('app2 must be here');
            expect(apps.get('key1-app2').last_modified).toBe('2017-06-11T10:20:22.874779+02:00', 'app2 have to be up to date');
        }).unsubscribe();
        expect(checkApp).toBe(true);

        appService.updateCache(pip1Update);
        appService.updateCache(pip2Update);
        appService.updateCache(pip3Update);

        // Check pipeline result
        let checkPip = false;
        pipelineStore.getPipelines('key').subscribe(pips => {
            checkPip = true;
            expect(pips.size).toBe(2, 'Must have 2 pipelines');
            expect(pips.get('key1-pip1')).toBeTruthy('pip1 must be here');
            expect(pips.get('key1-pip1').last_modified).toBe(1494490822, 'no change on pip1');
            expect(pips.get('key1-pip2')).toBeTruthy('pip2 must be here');
            expect(pips.get('key1-pip2').last_modified).toBe(1497169222, 'pip2 have to be up to date');
        }).unsubscribe();
        expect(checkPip).toBe(true);
    }));

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
