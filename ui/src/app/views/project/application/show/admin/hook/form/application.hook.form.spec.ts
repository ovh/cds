/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import { TranslateService, TranslateLoader} from 'ng2-translate/ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {ToasterService} from 'angular2-toaster/angular2-toaster';
import {ApplicationModule} from '../../../../application.module';
import {Project} from '../../../../../../../model/project.model';
import {Pipeline} from '../../../../../../../model/pipeline.model';
import {Application} from '../../../../../../../model/application.model';
import {SharedModule} from '../../../../../../../shared/shared.module';
import {ApplicationStore} from '../../../../../../../service/application/application.store';
import {ApplicationService} from '../../../../../../../service/application/application.service';
import {ToastService} from '../../../../../../../shared/toast/ToastService';
import {Observable} from 'rxjs';
import {ApplicationHookFormComponent} from './application.hook.form.component';
import {Hook} from '../../../../../../../model/hook.model';
import {TranslateParser} from 'ng2-translate';

describe('CDS: Application hook form component', () => {

    let injector: Injector;
    let toastService: ToastService;
    let appStore: ApplicationStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                { provide: ToastService, useClass: MockToast},
                TranslateLoader,
                ToasterService,
                TranslateService,
                { provide: ApplicationStore, useClass: MockAppStore},
                ApplicationService,
                TranslateParser,
            ],
            imports : [
                ApplicationModule,
                SharedModule,
                RouterTestingModule.withRoutes([]),

            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        appStore = injector.get(ApplicationStore);
        toastService = injector.get(ToastService);

    });

    afterEach(() => {
        injector = undefined;
        toastService = undefined;
        appStore = undefined;
    });


    it('should add a new hook', fakeAsync( () => {
        let call = 0;


        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ApplicationHookFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init Input
        let p: Project = new Project();
        p.key = 'key1';
        p.pipelines = new Array<Pipeline>();

        let pipeline: Pipeline = new Pipeline();
        pipeline.name = 'pipName';
        p.pipelines.push(pipeline);

        let a: Application = new Application();
        a.name = 'myApp';
        a.repositories_manager = { id: 1, name: 'repoman', type: 'typeR', url: 'foo.bar' };

        fixture.componentInstance.project = p;
        fixture.componentInstance.application = a;

        fixture.detectChanges();
        tick(50);

        fixture.componentInstance.ngOnInit();

        spyOn(toastService, 'success');

        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('button[name="addhookbtn"]').click();

        tick(50);

        expect(toastService.success).toHaveBeenCalled();


    }));
});

class MockAppStore {
   addHook(p: Project, a: Application, h: Hook) {
       return Observable.of({ 'a': 'a'});
   }
}

class MockToast {
    success(title: string, msg: string) {

    }
}

