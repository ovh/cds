import { HttpClient, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { ErrorHandler, LOCALE_ID, NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { TranslateLoader, TranslateModule } from '@ngx-translate/core';
import { TranslateHttpLoader } from '@ngx-translate/http-loader';
import { EventService } from 'app/event.service';
import { NgxsStoreModule } from 'app/store/store.module';
import * as Raven from 'raven-js';
import { AppComponent } from './app.component';
import { routing } from './app.routing';
import { AppService } from './app.service';
import { ServicesModule } from './service/services.module';
import { SharedModule } from './shared/shared.module';
import { NgxsStoragePluginModule, StorageEngine, STORAGE_ENGINE } from '@ngxs/storage-plugin';
import { PreferencesState } from './store/preferences.state';
import { EventV2Service } from './event-v2.service';
import { SearchComponent } from './views/search/search.component';
import { NavbarComponent } from './views/navbar/navbar.component';
import { HomeComponent } from './views/home/home.component';
import { SearchBarComponent } from './views/search/search-bar.component';

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
};

export class CDSStorageEngine implements StorageEngine {
    get length(): number { return localStorage.length }
    getItem(key: string): any { return localStorage.getItem(`CDS-${key.toUpperCase()}`) }
    setItem(key: string, val: any): void { return localStorage.setItem(`CDS-${key.toUpperCase()}`, val) }
    removeItem(key: string): void { localStorage.removeItem(`CDS-${key.toUpperCase()}`) }
    clear(): void { localStorage.clear() }
}

@NgModule({
    declarations: [
        AppComponent,
        HomeComponent,
        NavbarComponent,
        SearchBarComponent,
        SearchComponent
    ],
    imports: [
        BrowserModule,
        BrowserAnimationsModule,
        NgxsStoreModule,
        NgxsStoragePluginModule.forRoot({ keys: [PreferencesState] }),
        SharedModule,
        ServicesModule.forRoot(),
        routing,
        TranslateModule.forRoot({
            loader: {
                provide: TranslateLoader,
                useFactory: createTranslateLoader,
                deps: [HttpClient]
            }
        })
    ],
    exports: [
        ServicesModule,
    ],
    providers: [
        AppService,
        EventService,
        EventV2Service,
        { provide: ErrorHandler, useFactory: errorFactory },
        { provide: LOCALE_ID, useValue: 'en' },
        { provide: STORAGE_ENGINE, useClass: CDSStorageEngine },
        provideHttpClient(withInterceptorsFromDi())
    ],
    bootstrap: [AppComponent]
})
export class AppModule { }

export function createTranslateLoader(http: HttpClient) {
    return new TranslateHttpLoader(http, './assets/i18n/', '.json');
}
