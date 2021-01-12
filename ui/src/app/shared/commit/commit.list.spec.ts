/* eslint-disable @typescript-eslint/no-unused-vars */
import {TestBed, fakeAsync} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {APP_BASE_HREF} from '@angular/common';
import {SharedModule} from '../shared.module';
import {CommitListComponent} from './commit.list.component';

describe('CDS: Commit List', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
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
        }).compileComponents();
    });

    it('should load component', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(CommitListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
