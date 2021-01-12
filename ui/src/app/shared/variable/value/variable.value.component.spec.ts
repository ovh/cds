/* eslint-disable @typescript-eslint/no-unused-vars */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {VariableValueComponent} from '../value/variable.value.component';
import {SharedService} from '../../shared.service';
import {SharedModule} from '../../shared.module';

describe('CDS: Variable Value Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
            ],
            imports : [
                SharedModule
            ]
        }).compileComponents();
    });

    it('Load Component string', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(VariableValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'string';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('input[type=text]')).toBeTruthy('INput type text must be displayed');
    }));

    it('Load Component password', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(VariableValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'password';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('input[type=password]')).toBeTruthy('Input type password must be displayed');
    }));

    it('Load Component text', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(VariableValueComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.type = 'text';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('textarea')).toBeTruthy('textarea must be displayed');
    }));
});

