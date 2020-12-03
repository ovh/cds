import { APP_BASE_HREF } from '@angular/common';
import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { fakeAsync, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Parameter } from 'app/model/parameter.model';
import { ParameterService } from 'app/service/parameter/parameter.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { SharedModule } from '../../shared.module';
import { ParameterEvent } from '../parameter.event.model';
import { ParameterListComponent } from './parameter.component';

describe('CDS: Parameter List Component', () => {
    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ParameterService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                RepoManagerService,
                { provide: APP_BASE_HREF, useValue: '/' },
                ThemeStore
            ],
            imports: [
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        }).compileComponents();
    });


    it('should load component + update description', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);

        let typeMock = ['string', 'password'];

        // Create component
        let fixture = TestBed.createComponent(ParameterListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne(((req: HttpRequest<any>) => req.url === '/parameter/type')).flush(typeMock);

        let params: Parameter[] = [];
        let p: Parameter = new Parameter();
        p.name = 'foo';
        p.type = 'string';
        p.description = 'foo is my param';

        params.push(p);
        fixture.componentInstance.parameters = params;

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
            new ParameterEvent('delete', fixture.componentInstance.parameters[0])
        );

        http.verify();
    }));
});

