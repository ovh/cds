/* eslint-disable @typescript-eslint/no-unused-vars */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
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

        let compiled = fixture.debugElement.nativeElement;

        let r = new Requirement('binary');
        r.name = 'foo';
        r.value = 'foo';

        fixture.detectChanges();
        tick(250);

        // simulate typing new variable
        let inputName = compiled.querySelector('input[name="value"]');
        inputName.value = r.value;
        inputName.dispatchEvent(new Event('input'));
        inputName.dispatchEvent(new Event('keyup'));

        fixture.detectChanges();
        tick(250);

        spyOn(fixture.componentInstance.event, 'emit');
        compiled.querySelector('.ui.blue.button').click();

        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(new RequirementEvent('add', r));
    }));
});
