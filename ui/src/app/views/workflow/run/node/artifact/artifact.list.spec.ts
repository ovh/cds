/* eslint-disable @typescript-eslint/no-unused-vars */
import {TestBed, fakeAsync} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateLoader, TranslateModule, TranslateParser, TranslateService} from '@ngx-translate/core';
import {SharedModule} from 'app/shared/shared.module';
import {WorkflowModule} from 'app/views/workflow/workflow.module';
import {WorkflowRunArtifactListComponent} from './artifact.list.component';

describe('CDS: Artifact List', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
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
        }).compileComponents();
    });

    it('should load component', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(WorkflowRunArtifactListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
