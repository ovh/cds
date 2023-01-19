import { APP_BASE_HREF } from '@angular/common';
import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { fakeAsync, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Parameter } from 'app/model/parameter.model';
import { ParameterService } from 'app/service/parameter/parameter.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { SharedModule } from '../../shared.module';
import { ParameterListComponent } from './parameter.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

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
                { provide: APP_BASE_HREF, useValue: '/' }
            ],
            imports: [
                RouterTestingModule.withRoutes([]),
                SharedModule,
                BrowserAnimationsModule,
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
        http.verify();
    }));
});

