/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {SharedService} from '../../shared.service';
import {ParameterValueComponent} from './parameter.value.component';
import {SharedModule} from '../../shared.module';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Parameter Value Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                RepoManagerService,
            ],
            imports : [
                SharedModule,
                HttpClientTestingModule
            ]
        });
    });

    it('should create an input text', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ParameterValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'string';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('input[type=text]')).toBeTruthy('INput type text must be displayed');
    }));

    it('should create an input number', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ParameterValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'number';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('input[type=number]')).toBeTruthy('Input type number must be displayed');
    }));

    it('should create a checkbox', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ParameterValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'boolean';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('input[type=checkbox]')).toBeTruthy('Input type checkbox must be displayed');
    }));

    /*
    it('should create a textarea', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ParameterValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'text';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('codemirror')).toBeTruthy('textarea must be displayed');

    }));
    */

    it('should create a select for pipeline', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ParameterValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'pipeline';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('select')).toBeTruthy('select must be displayed');
    }));

    it('should create a select for environments', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ParameterValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'env';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('select')).toBeTruthy('select must be displayed');
    }));

});

