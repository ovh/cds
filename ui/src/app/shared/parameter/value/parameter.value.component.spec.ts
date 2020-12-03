import { HttpClientTestingModule } from '@angular/common/http/testing';
import { fakeAsync, TestBed, tick } from '@angular/core/testing';
import { ThemeStore } from 'app/service/theme/theme.store';
import { RepoManagerService } from '../../../service/repomanager/project.repomanager.service';
import { SharedModule } from '../../shared.module';
import { SharedService } from '../../shared.service';
import { ParameterValueComponent } from './parameter.value.component';

describe('CDS: Parameter Value Component', () => {
    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                RepoManagerService,
                ThemeStore,
            ],
            imports: [
                SharedModule,
                HttpClientTestingModule
            ]
        }).compileComponents();
    });

    it('should create an input text', fakeAsync(() => {
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

    it('should create an input number', fakeAsync(() => {
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

    it('should create a checkbox', fakeAsync(() => {
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

    it('should create a select for pipeline', fakeAsync(() => {
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

    it('should create a select for environments', fakeAsync(() => {
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

