/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {ActivatedRoute, Router} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {Observable} from 'rxjs/Observable';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {PipelineService} from '../../../../service/pipeline/pipeline.service';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {ToastService} from '../../../../shared/toast/ToastService';
import {PipelineModule} from '../../pipeline.module';
import {SharedModule} from '../../../../shared/shared.module';
import {PipelineAdminComponent} from './pipeline.admin.component';
import {Project} from '../../../../model/project.model';
import {Pipeline} from '../../../../model/pipeline.model';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import 'rxjs/add/observable/of';

describe('CDS: Pipeline Admin Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                MockBackend,
                {provide: XHRBackend, useClass: MockBackend},
                PipelineService,
                PipelineStore,
                {provide: ActivatedRoute, useClass: MockActivatedRoutes},
                {provide: ToastService, useClass: MockToast},
                { provide: Router, useClass: MockRouter},
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports: [
                PipelineModule,
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

        let pipStore: PipelineStore = injector.get(PipelineStore);
        spyOn(pipStore, 'updatePipeline').and.callFake(() => {
            return Observable.of(pip);
        });
        fixture.debugElement.nativeElement.querySelector('.ui.button.green.button').click();

        expect(pipStore.updatePipeline).toHaveBeenCalledTimes(1);
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({key: 'key1', pipName: 'pip1'});
        this.queryParams = Observable.of({key: 'key1', appName: 'pip1', tab: 'workflow'});
    }
}

class MockRouter {
    public navigate() {
    }
}
