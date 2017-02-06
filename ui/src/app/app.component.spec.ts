/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed} from '@angular/core/testing';
import {AppModule} from './app.module';
import {AppComponent} from './app.component';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend, MockConnection} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {AuthentificationStore} from './service/auth/authentification.store';
import {User} from './model/user.model';
import {SharedModule} from './shared/shared.module';
import {TranslateService, TranslateParser} from 'ng2-translate';

describe('App: CDS', () => {

    let injector: Injector;
    let backend: MockBackend;
    let connection: MockConnection;
    let authStore: AuthentificationStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                TranslateService,
                TranslateParser,
            ],
            imports : [
                AppModule,
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

        injector = getTestBed();
        backend = injector.get(XHRBackend);
        backend.connections.subscribe((c: MockConnection) => connection = c);
        authStore = injector.get(AuthentificationStore);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        connection = undefined;
        authStore = undefined;
    });


    it('should create the app', async( () => {
        let fixture = TestBed.createComponent(AppComponent);
        let app = fixture.debugElement.componentInstance;
        expect(app).toBeTruthy();
    }));


    it('should render a navbar', async(() => {
        let fixture = TestBed.createComponent(AppComponent);
        let compiled = fixture.debugElement.nativeElement;

        expect(fixture.componentInstance.isConnected).toBeFalsy('IsConnected flag must be false');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must not have the css class "connected"');

        fixture.componentInstance.ngOnInit();
        authStore.addUser(new User(), false);

        expect(fixture.componentInstance.isConnected).toBeTruthy('IsConnected flag must be true');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must have connected css class');
    }));

});
