import {HttpClient, HttpClientModule} from '@angular/common/http';
import {LOCALE_ID, NgModule} from '@angular/core';
import {BrowserModule} from '@angular/platform-browser';
import {TranslateLoader, TranslateModule} from '@ngx-translate/core';
import {TranslateHttpLoader} from '@ngx-translate/http-loader';
import {ToasterModule} from 'angular2-toaster/angular2-toaster';
import {AppComponent} from './app.component';
import {routing} from './app.routing';
import {AppService} from './app.service';
import {ServicesModule} from './service/services.module';
import {SharedModule} from './shared/shared.module';
import {NavbarModule} from './views/navbar/navbar.module';

@NgModule({
    declarations: [
        AppComponent
    ],
    imports: [
        BrowserModule,
        HttpClientModule,
        NavbarModule,
        SharedModule,
        ServicesModule.forRoot(),
        routing,
        ToasterModule,
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
        { provide: LOCALE_ID, useValue: navigator.language.match(/fr/) ? 'fr' : 'en' }
    ],
    bootstrap: [AppComponent]
})
export class AppModule {
}

export function createTranslateLoader(http: HttpClient) {
    return new TranslateHttpLoader(http, 'assets/i18n/', '.json');
}
