/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {RepoManagerService} from '../../../../../../service/repomanager/project.repomanager.service';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ProjectService} from '../../../../../../service/project/project.service';
import {PipelineService} from '../../../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../../../service/environment/environment.service';
import {VariableService} from '../../../../../../service/variable/variable.service';
import {SharedModule} from '../../../../../../shared/shared.module';
import {ToasterService} from 'angular2-toaster/angular2-toaster';
import {Project} from '../../../../../../model/project.model';
import {TranslateParser} from '@ngx-translate/core';
import {ProjectModule} from '../../../../project.module';
import {ProjectRepoManagerComponent} from './project.repomanager.list.component';
import {RepositoriesManager} from '../../../../../../model/repositories.model';
import {Observable} from 'rxjs/Observable';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import 'rxjs/add/observable/of';
describe('CDS: Project RepoManager List Component', () => {

    let injector: Injector;
    let backend: MockBackend;
    let projectStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                ToasterService,
                TranslateService,
                TranslateParser,
                { provide: ToastService, useClass: MockToast}
            ],
            imports : [
                ProjectModule,
                SharedModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        projectStore = undefined;
    });


    it('it should delete a repo manager', fakeAsync( () => {
        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectRepoManagerComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let p: Project = new Project();
        p.key = 'key1';
        fixture.componentInstance.project = p;

        let reposMans = new Array<RepositoriesManager>();
        let r: RepositoriesManager = { name : 'stash'};
        reposMans.push(r);
        fixture.componentInstance.reposmanagers = reposMans;

        fixture.detectChanges();
        tick(250);

        spyOn(projectStore, 'disconnectRepoManager').and.callFake(() => {
            return Observable.of(p);
        });

        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges();
        tick(50);
        compiled.querySelector('.ui.red.button.active').click();

        expect(projectStore.disconnectRepoManager).toHaveBeenCalledWith('key1', 'stash');
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}
