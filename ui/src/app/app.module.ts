import {BrowserModule} from '@angular/platform-browser';
import {NgModule, LOCALE_ID} from '@angular/core';
import {AppComponent} from './app.component';
import {ServicesModule} from './service/services.module';
import {TranslateModule, TranslateLoader} from '@ngx-translate/core';
import {TranslateHttpLoader} from '@ngx-translate/http-loader';
import {routing} from './app.routing';
import {NavbarModule} from './views/navbar/navbar.module';
import {SharedModule} from './shared/shared.module';
import {ToasterModule} from 'angular2-toaster/angular2-toaster';
import {AppService} from './app.service';
import {HttpClientModule, HttpClient} from '@angular/common/http';

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
