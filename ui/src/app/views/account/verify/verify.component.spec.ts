/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, fakeAsync, tick, inject} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';

import {UserService} from '../../../service/user/user.service';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {AppModule} from '../../../app.module';
import {Router, ActivatedRoute} from '@angular/router';
import {VerifyComponent} from './verify.component';
import {User} from '../../../model/user.model';
import {AccountModule} from '../account.module';

describe('CDS: VerifyComponent', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                { provide: XHRBackend, useClass: MockBackend },
                UserService,
                AuthentificationStore,
                { provide: Router, useClass: MockRouter},
                { provide: ActivatedRoute, useClass: MockActivatedRoutes}
            ],
            imports : [
                AppModule,
                RouterTestingModule.withRoutes([]),
                AccountModule
            ]
        });
    });

    it('Verify OK', fakeAsync( inject([XHRBackend], (backend: MockBackend) => {
        // Create component
        let fixture = TestBed.createComponent(VerifyComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Mock Http login request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '{ "user": { "username": "foo" }, "password": "bar"}'})));
        });

        fixture.detectChanges();
        tick(50);

        // Init verify account
        expect(backend.connectionsArray.length).toBe(1);
        expect(backend.connectionsArray[0].request.url).toBe('foo.bar/user/foo/confirm/myToken',
            'Url to API is wrong. Must be /user/:username/confirm/:token');
        expect(fixture.componentInstance.showErrorMessage).toBeFalsy('We must not show error message is activation is ok');

        // Then SignIn
        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('#loginButton').click();

        // Check api call
        expect(backend.connectionsArray.length).toBe(2);
        expect(backend.connectionsArray[1].request.url).toBe('foo.bar/login', 'API login handler must be call');

        // check body
        let userSent: User = JSON.parse(backend.connectionsArray[1].request.getBody());
        expect(userSent.username).toBe('foo');
        expect(userSent.password).toBe('bar');

    })));
});

export class MockRouter {
    public navigate() {
    }
}

export class MockActivatedRoutes {
    snapshot = {
        params: {
            'username': 'foo',
            'token': 'myToken'
        }
    };
}
