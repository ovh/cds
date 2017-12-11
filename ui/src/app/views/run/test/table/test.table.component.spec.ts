/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {ApplicationRunModule} from '../../application.run.module';
import {SharedModule} from '../../../../shared/shared.module';
import {TestTableComponent} from './test.table.component';
import {TestSuite, TestCase, Failure} from '../../../../model/pipeline.model';

describe('CDS: Test table component', () => {

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

    it('should load component + filter testcase', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(TestTableComponent);
        let component = fixture.debugElement.componentInstance;

        fixture.componentInstance.tests = new Array<TestSuite>();
        fixture.componentInstance.tests.push(createTestSuite(false, true));
        fixture.componentInstance.tests.push(createTestSuite(true, false));
        fixture.componentInstance.tests.push(createTestSuite(false, false));

        fixture.componentInstance.filter = 'error';

        fixture.componentInstance.updateFilteredTests();

        expect(component).toBeTruthy();
        expect(fixture.componentInstance.filteredTests.length).toBe(2);
    }));
});

function createTestSuite(error: boolean, failure: boolean): TestSuite {
    let ts = new TestSuite();
    let tc = new TestCase();
    if (error) {
        ts.errors = 1;
        tc.errors = new Array<Failure>();
        let f = new Failure();
        f.value = 'my error';
        tc.errors.push(f);
    }
    if (failure) {
        ts.failures = 1;
        tc.failures = new Array<Failure>();
        let f = new Failure();
        f.value = 'my error';
        tc.failures.push(f);
    }
    ts.tests = new Array<TestCase>();
    ts.tests.push(tc);
    return ts;
}
