/* tslint:disable:no-unused-variable */
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { Injector } from '@angular/core';
import { fakeAsync, getTestBed, TestBed, tick } from '@angular/core/testing';
import { XHRBackend } from '@angular/http';
import { MockBackend } from '@angular/http/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { NgxsStoreModule } from 'app/store/store.module';
import 'rxjs/add/observable/of';
import { Observable } from 'rxjs/Observable';
import { Pipeline } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { PipelineService } from '../../../../service/pipeline/pipeline.service';
import { SharedModule } from '../../../../shared/shared.module';
import { ToastService } from '../../../../shared/toast/ToastService';
import { PipelineModule } from '../../pipeline.module';
import { PipelineAdminComponent } from './pipeline.admin.component';

describe('CDS: Pipeline Admin Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                PipelineService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                { provide: ToastService, useClass: MockToast },
                { provide: Router, useClass: MockRouter },
                NavbarService,
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports: [
                PipelineModule,
                NgxsStoreModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(MockBackend);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });

    it('should update pipeline', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(PipelineAdminComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let p: Project = new Project();
        p.key = 'key1';
        fixture.componentInstance.project = p;

        let pip: Pipeline = new Pipeline();
        pip.name = 'pipName';
        fixture.componentInstance.pipeline = pip;


        fixture.detectChanges();
        tick(250);

        let store: Store = injector.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => {
            return Observable.of(null);
        });
        fixture.debugElement.nativeElement.querySelector('.ui.button.green.button').click();

        expect(store.dispatch).toHaveBeenCalledTimes(1);
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({ key: 'key1', pipName: 'pip1' });
        this.queryParams = Observable.of({ key: 'key1', appName: 'pip1', tab: 'workflow' });
    }
}

class MockRouter {
    public navigate() {
    }
}
