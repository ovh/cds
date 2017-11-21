/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../../shared.module';
import {PrerequisitesFormComponent} from './prerequisites.form.component';
import {Prerequisite} from '../../../model/prerequisite.model';
import {PrerequisiteEvent} from '../prerequisite.event.model';

describe('CDS: prerequisite From Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

    });


    it('should create a new prerequisite', fakeAsync( () => {

        // Create component
        let fixture = TestBed.createComponent(PrerequisitesFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        let pre = new Prerequisite();
        pre.parameter = 'foo';
        pre.expected_value = 'bar';

        fixture.detectChanges();
        tick(50);

        // simulate typing new parameter
        let inputName = compiled.querySelector('input[name="value"]');
        inputName.value = pre.expected_value;
        inputName.dispatchEvent(new Event('input'));

        fixture.componentInstance.newPrerequisite.parameter = '';

        fixture.componentInstance.prerequisites = [
            { parameter: 'git.branch', expected_value: '' }
        ];

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.event, 'emit');
        compiled.querySelector('.ui.blue.button').click();

        expect(fixture.componentInstance.event.emit).not.toHaveBeenCalled();

        fixture.componentInstance.newPrerequisite.parameter = 'foo';
        compiled.querySelector('.ui.blue.button').click();

        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(new PrerequisiteEvent('add', pre));
    }));
});

