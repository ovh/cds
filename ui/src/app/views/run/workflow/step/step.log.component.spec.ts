/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {TranslateService, TranslateParser, TranslateLoader} from 'ng2-translate';
import {ApplicationPipelineService} from '../../../../service/application/pipeline/application.pipeline.service';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {ApplicationRunModule} from '../../application.run.module';
import {SharedModule} from '../../../../shared/shared.module';
import {DurationService} from '../../../../shared/duration/duration.service';
import {StepLogComponent} from './step.log.component';

describe('App: CDS', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                TranslateService,
                TranslateParser,
                ApplicationPipelineService,
                AuthentificationStore,
                TranslateLoader,
                TranslateService,
                DurationService,
                TranslateParser
            ],
            imports : [
                ApplicationRunModule,
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });


    it('should load component', async( () => {
        let fixture = TestBed.createComponent(StepLogComponent);
        let app = fixture.debugElement.componentInstance;
        expect(app).toBeTruthy();
    }));
});
