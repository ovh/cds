import { fakeAsync, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { SharedModule } from 'app/shared/shared.module';
import { GroupFormComponent } from './group.form.component';

describe('CDS: Group form component', () => {
    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
            ],
            imports: [
                SharedModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([])
            ]
        }).compileComponents();

    });

    it('should load component and disable button', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(GroupFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(250);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.green.button.disabled')).toBeTruthy();
    }));

    it('should load component and enable button', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(GroupFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.group.name = 'My New Group';

        fixture.detectChanges();
        tick(250);
        expect(fixture.debugElement.nativeElement.querySelector('.ui.green.button.disabled')).toBeFalsy();
    }));
});

