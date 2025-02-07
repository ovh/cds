import { provideHttpClientTesting } from '@angular/common/http/testing';
import { fakeAsync, flush, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Parameter } from 'app/model/parameter.model';
import { ParameterService } from 'app/service/parameter/parameter.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { Observable, of } from 'rxjs';
import { SharedModule } from '../../shared.module';
import { ParameterEvent } from '../parameter.event.model';
import { ParameterFormComponent } from './parameter.form';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgxsModule } from '@ngxs/store';
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { PreferencesState } from 'app/store/preferences.state';

describe('CDS: parameter From Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: ParameterService, useClass: MockParameterService },
                ParameterService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                RepoManagerService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting()
            ],
            imports: [
                NgxsModule.forRoot([
                    PreferencesState
                ]),
                SharedModule,
                BrowserAnimationsModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([])
            ]
        }).compileComponents();
    });


    it('should create a new parameter', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ParameterFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        let parameter = new Parameter();
        parameter.name = 'foo';
        parameter.type = 'string';
        parameter.description = 'bar';

        fixture.detectChanges();
        tick(50);

        // simulate typing new parameter
        let inputName = compiled.querySelector('input[name="name"]');
        inputName.value = parameter.name;
        inputName.dispatchEvent(new Event('input'));

        fixture.componentInstance.newParameter.type = 'string';

        let textAreaDescription = compiled.querySelector('textarea[name="description"]');
        textAreaDescription.value = parameter.description;
        textAreaDescription.dispatchEvent(new Event('input'));
        textAreaDescription.dispatchEvent(new Event('change'));

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.createParameterEvent, 'emit');
        compiled.querySelector('button[name="addBtn"]').click();

        parameter.value = '';
        expect(fixture.componentInstance.createParameterEvent.emit).toHaveBeenCalledWith(new ParameterEvent('add', parameter));

        flush();
    }));
});

class MockParameterService {
    constructor() { }

    getTypesFromCache(): string[] {
        return [];
    }

    getTypesFromAPI(): Observable<string[]> {
        return of(['string', 'password']);
    }
}
