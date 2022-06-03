/* eslint-disable @typescript-eslint/no-unused-vars */

import {TestBed, fakeAsync, tick} from '@angular/core/testing';
import {TranslateService, TranslateParser, TranslateLoader, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {APP_BASE_HREF} from '@angular/common';
import {SharedService} from '../../shared.service';
import {ParameterService} from '../../../service/parameter/parameter.service';
import {SharedModule} from '../../shared.module';
import {Action} from '../../../model/action.model';
import {ActionStepComponent} from './step.component';
import {StepEvent} from './step.event';


describe('CDS: Step Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
                ParameterService,
                TranslateParser,
                TranslateLoader,
                { provide: APP_BASE_HREF , useValue : '/' }
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                TranslateModule.forRoot(),
                SharedModule
            ]
        }).compileComponents();
    });
});
