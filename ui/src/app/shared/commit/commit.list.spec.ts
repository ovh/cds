/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {CommitListComponent} from './commit.list.component';
import {SharedModule} from '../shared.module';

describe('CDS: Commit List', () => {

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
                RouterTestingModule.withRoutes([]),
                TranslateModule.forRoot(),
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
        let fixture = TestBed.createComponent(CommitListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
