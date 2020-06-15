/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {CommitListComponent} from './commit.list.component';
import {SharedModule} from '../shared.module';
import {APP_BASE_HREF} from '@angular/common';

describe('CDS: Commit List', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
                { provide: APP_BASE_HREF, useValue : '/' }
            ],
            imports: [
                RouterTestingModule.withRoutes([]),
                TranslateModule.forRoot(),
                SharedModule
            ]
        });
    });

    it('should load component', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(CommitListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
