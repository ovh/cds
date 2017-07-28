/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {MockBackend} from '@angular/http/testing';
import {Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {AppModule} from '../../app.module';
import {AuthentificationStore} from '../auth/authentification.store';
import {User} from '../../model/user.model';
import {UserService} from './user.service';
import {RouterModule} from '@angular/router';

describe('CDS: User Service + Authent Store', () => {

    let injector: Injector;
    let backendUser: MockBackend;
    let authenStore: AuthentificationStore;
    let userService: UserService;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                MockBackend,
                AuthentificationStore,
                UserService
            ],
            imports : [
                AppModule,
                RouterModule
            ]
        });
        injector = getTestBed();
        backendUser = injector.get(MockBackend);
        authenStore = injector.get(AuthentificationStore);
        userService = injector.get(UserService);

    });

    afterEach(() => {
        injector = undefined;
        backendUser = undefined;
        authenStore = undefined;
        userService = undefined;
    });

    it('login', async( () => {
        let connectedChecked = false;
        let started = false;

        // user to create
        let u = new User();
        u.id = 1;
        u.username = 'foo';

        // Mock Http login request
        backendUser.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({body: '{ "user": { "id": 1, "username": "foo" } }'})));
        });

        // Assertion
        authenStore.getUserlst().subscribe( user => {
            if (started) {
                connectedChecked = true;
                expect(user.id).toBe(1, 'Wrong user id');
                expect(authenStore.isConnected).toBeTruthy('User must be connected');
            }
        });

        // Begin test
        started = true;
        userService.login(u).subscribe( () => {});

        // Final assertion
        expect(connectedChecked).toBeTruthy('User never connected');

    }));
});
