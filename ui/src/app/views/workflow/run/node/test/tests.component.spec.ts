/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {WorkflowRunTestsResultComponent} from './tests.component';
import {WorkflowModule} from '../../../workflow.module';
import {SharedModule} from '../../../../../shared/shared.module';

describe('CDS: Test Report component', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
            ],
            imports: [
                WorkflowModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });

    it('should load component', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(WorkflowRunTestsResultComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
