/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync, inject} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {LoginComponent} from './login.component';

import {UserService} from '../../../service/user/user.service';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {AppModule} from '../../../app.module';
import {User} from '../../../model/user.model';
import {Router} from '@angular/router';
import {AccountModule} from '../account.module';

describe('CDS: LoginComponent', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                { provide: XHRBackend, useClass: MockBackend },
                UserService,
                AuthentificationStore,
                { provide: Router, useClass: MockRouter}
            ],
            imports : [
                AppModule,
                RouterTestingModule.withRoutes([]),
                AccountModule
            ]
        });
    });


    it('Click on Login button', fakeAsync(  inject([XHRBackend], (backend: MockBackend) => {
        // Create loginComponent
        let fixture = TestBed.createComponent(LoginComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Mock Http login request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '{ "user": { "username": "foo" } }'})));
        });


        let compiled = fixture.debugElement.nativeElement;

        // Start detecting change in model
        fixture.detectChanges();
        tick(50);

        // Simulate user typing
        let inputUsername = compiled.querySelector('input[name="username"]');
        inputUsername.value = 'foo';
        inputUsername.dispatchEvent(new Event('input'));

        let inputPassword = compiled.querySelector('input[name="password"]');
        inputPassword.value = 'bar';
        inputPassword.dispatchEvent(new Event('input'));

        // Simulate user click
        compiled.querySelector('#loginButton').click();
        expect(backend.connectionsArray.length).toBe(1);
        let userSent: User = JSON.parse(backend.connectionsArray[0].request.getBody());
        expect(userSent.username).toBe('foo');
        expect(userSent.password).toBe('bar');

    })));
});

export class MockRouter {
    public navigate() {
    }
}
