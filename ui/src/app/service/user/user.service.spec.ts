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
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';

describe('CDS: User Service + Authent Store', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                AuthentificationStore,
                UserService
            ],
            imports : [
                AppModule,
                RouterModule,
                HttpClientTestingModule
            ]
        });

    });

    it('login', async( () => {
        const http = TestBed.get(HttpTestingController);
        let connectedChecked = false;
        let started = false;

        // user to create
        let u = new User();
        u.id = 1;
        u.username = 'foo';

        let loginResponse = {
            'user': {
                'id': 1,
                'username': 'foo'
            }
        };

        let authenStore = TestBed.get(AuthentificationStore);
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
        let userService = TestBed.get(UserService);
        userService.login(u).subscribe( () => {});
        http.expectOne('http://localhost:80801/login').flush(loginResponse);


        // Final assertion
        expect(connectedChecked).toBeTruthy('User never connected');

    }));
});
