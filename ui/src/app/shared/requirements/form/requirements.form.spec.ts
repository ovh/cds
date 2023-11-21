import { TestBed, tick, fakeAsync, flush } from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {WorkerModelService} from '../../../service/worker-model/worker-model.service';
import {SharedModule} from '../../shared.module';
import {RequirementService} from '../../../service/requirement/requirement.service';
import {RequirementStore} from '../../../service/requirement/requirement.store';
import {RequirementsFormComponent} from './requirements.form.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

describe('CDS: Requirement Form Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                RequirementService,
                RequirementStore,
                TranslateService,
                WorkerModelService,
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                SharedModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                BrowserAnimationsModule,
                HttpClientTestingModule
            ]
        }).compileComponents();
    });

    it('should create a new requirement and auto write name', fakeAsync( () => {
        const http = TestBed.get(HttpTestingController);


        // Create component
        let fixture = TestBed.createComponent(RequirementsFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne('/requirement/types').flush(['binary']);

        let r = new Requirement('binary');
        r.name = 'foo';
        r.value = 'foo';

        fixture.componentInstance.newRequirement = r;

        tick(250)
        fixture.detectChanges();

        let compiled = fixture.debugElement.nativeElement;
        spyOn(fixture.componentInstance.event, 'emit');
        compiled.querySelector('button[name="addBtn"]').click();

        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(new RequirementEvent('add', r));

        flush();
    }));
});
