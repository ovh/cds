import {TestBed, fakeAsync} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {WorkflowModule} from '../../../workflow.module';
import {SharedModule} from '../../../../../shared/shared.module';
import {WorkflowRunTestsResultComponent} from './tests.component';
import {PluginManager} from "@ngxs/store/src/plugin-manager";
import {NgxsStoragePlugin} from "@ngxs/storage-plugin";
import {NgxsModule} from "@ngxs/store";
import {NgxsTestModule} from "@ngxs/store/internals/testing/helpers/ngxs-test.module";

describe('CDS: Test Report component', () => {

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
                SharedModule,
                NgxsModule.forRoot()
            ]
        }).compileComponents();
    });

    it('should load component', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(WorkflowRunTestsResultComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
