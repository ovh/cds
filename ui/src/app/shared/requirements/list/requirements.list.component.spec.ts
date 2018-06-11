/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../../shared.module';
import {RequirementsListComponent} from './requirements.list.component';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {WorkerModelService} from '../../../service/worker-model/worker-model.service';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {RequirementStore} from '../../../service/requirement/requirement.store';
import {RequirementService} from '../../../service/requirement/requirement.service';

describe('CDS: Requirement List Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateParser,
                RequirementService,
                TranslateService,
                RequirementStore,
                WorkerModelService,
                TranslateLoader
            ],
            imports : [
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });
    });


    it('should load component + delete requirement', fakeAsync(  () => {
        const http = TestBed.get(HttpTestingController);
        let mock = ['binary', 'network'];


        // Create component
        let fixture = TestBed.createComponent(RequirementsListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne('/requirement/types').flush(mock);


        expect(JSON.stringify(fixture.componentInstance.availableRequirements)).toBe(JSON.stringify(['binary', 'network']));

        let reqs: Requirement[] = [];
        let r: Requirement = new Requirement('binary');
        r.name = 'foo';
        r.value = 'bar';

        reqs.push(r);
        fixture.componentInstance.requirements = reqs;

        // Readonly mode: no delete button displayed
        expect(fixture.debugElement.nativeElement.querySelector('.ui.red.button')).toBeFalsy();

        fixture.componentInstance.edit = true;

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.ui.red.button')).toBeTruthy('Delete button must be displayed');
        compiled.querySelector('.ui.red.button').click();

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.event, 'emit');

        expect(compiled.querySelector('.ui.buttons')).toBeTruthy('Confirmation buttons must be displayed');
        compiled.querySelector('.ui.red.button.active').click();

        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(
            new RequirementEvent('delete', fixture.componentInstance.requirements[0])
        );
    }));
});
