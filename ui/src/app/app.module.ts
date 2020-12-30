import { HttpClient, HttpClientModule } from '@angular/common/http';
import { ErrorHandler, LOCALE_ID, NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { TranslateLoader, TranslateModule } from '@ngx-translate/core';
import { TranslateHttpLoader } from '@ngx-translate/http-loader';
import { ToasterModule } from 'angular2-toaster-sgu';
import { EventService } from 'app/event.service';
import { NgxsStoreModule } from 'app/store/store.module';
import * as Raven from 'raven-js';
import { AppComponent } from './app.component';
import { routing } from './app.routing';
import { AppService } from './app.service';
import { ServicesModule } from './service/services.module';
import { SharedModule } from './shared/shared.module';
import { NavbarModule } from './views/navbar/navbar.module';

export let errorFactory = () => {
    if ((<any>window).cds_sentry_url) {
        class RavenErrorHandler implements ErrorHandler {
            handleError(err: any): void {
                console.error(err);
                Raven.captureException(err);
            }
        }

        let tags = {};
        let username = localStorage.getItem('CDS-USER');
        if (username) {
            tags['CDS_USER'] = username;
        }

        Raven
            .config((<any>window).cds_sentry_url, { release: (<any>window).cds_version, tags })
            .install();

        return new RavenErrorHandler();
    } else {
        return new ErrorHandler();
    }
}


@NgModule({
    declarations: [
        AppComponent
    ],
    imports: [
        BrowserModule,
        BrowserAnimationsModule,
        HttpClientModule,
        NavbarModule,
        NgxsStoreModule,
        ToasterModule.forRoot(),
        SharedModule,
        ServicesModule.forRoot(),
        routing,
        TranslateModule.forRoot({
            loader: {
                provide: TranslateLoader,
                useFactory: createTranslateLoader,
                deps: [HttpClient]
            }
        }),
    ],
    exports: [
        ServicesModule,
    ],
    providers: [
        AppService,
        EventService,
        { provide: ErrorHandler, useFactory: errorFactory },
        { provide: LOCALE_ID, useValue: 'en' }
    ],
    bootstrap: [AppComponent]
})
export class AppModule {
}

export function createTranslateLoader(http: HttpClient) {
    return new TranslateHttpLoader(http, './assets/i18n/', '.json');
}
