/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {TestsResultComponent} from './tests.component';
import {ApplicationRunModule} from '../application.run.module';
import {SharedModule} from '../../../shared/shared.module';

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
                ApplicationRunModule,
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
        let fixture = TestBed.createComponent(TestsResultComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
