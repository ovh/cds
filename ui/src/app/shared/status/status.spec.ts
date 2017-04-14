/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {SharedService} from '../shared.service';
import {SharedModule} from '../shared.module';
import {StatusIconComponent} from './status.component';

describe('CDS: Parameter Value Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
            ],
            imports : [
                SharedModule
            ]
        });
    });

    it('should display success icon', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(StatusIconComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.status = 'Success';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.green.check.icon')).toBeTruthy('Success icon not displayed');
    }));

    it('should display building loader', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(StatusIconComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.status = 'Building';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.blue.loader')).toBeTruthy('Building loader not displayed');
    }));

    it('should display fail icon', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(StatusIconComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.status = 'Fail';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.remove.red.icon')).toBeTruthy('Fail icon not displayed');
    }));

    it('should display disabled icon', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(StatusIconComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.status = 'Disabled';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.ban.grey.icon')).toBeTruthy('Disabled icon not displayed');
    }));

    it('should display skipped icon', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(StatusIconComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.status = 'Skipped';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.ban.grey.icon')).toBeTruthy('Skipped icon not displayed');
    }));

    it('should display waiting icon', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(StatusIconComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.status = 'Waiting';

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.wait.blue.icon')).toBeTruthy('Waiting icon not displayed');
    }));


});

