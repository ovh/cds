/* eslint-disable @typescript-eslint/no-unused-vars */
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { NgxsStoreModule } from 'app/store/store.module';
import 'rxjs/add/observable/of';
import { Observable } from 'rxjs/Observable';
import { ApplicationService } from 'app/service/application/application.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { SharedModule } from '../../../../../../shared/shared.module';
import { PipelineModule } from '../../../../pipeline.module';


describe('CDS: Stage From component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                NavbarService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                WorkflowService,
                ApplicationService,
                EnvironmentService,
                WorkflowRunService
            ],
            imports: [
                PipelineModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
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
        this.params = Observable.of({ key: 'key1', appName: 'app1' });
        this.queryParams = Observable.of({ key: 'key1', appName: 'app1' });
    }
}
