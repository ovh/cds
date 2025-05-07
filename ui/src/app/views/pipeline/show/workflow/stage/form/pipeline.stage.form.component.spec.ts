import { provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { NgxsStoreModule } from 'app/store/store.module';
import { ApplicationService } from 'app/service/application/application.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { SharedModule } from '../../../../../../shared/shared.module';
import { PipelineModule } from '../../../../pipeline.module';
import { of } from 'rxjs';
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';


describe('CDS: Stage From component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                TranslateService,
                TranslateLoader,
                TranslateParser,
                WorkflowService,
                ApplicationService,
                EnvironmentService,
                WorkflowRunService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting()
            ],
            imports: [
                PipelineModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        }).compileComponents();
    });
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = of({ key: 'key1', appName: 'app1' });
        this.queryParams = of({ key: 'key1', appName: 'app1' });
    }
}
