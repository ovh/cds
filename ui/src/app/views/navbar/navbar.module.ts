import {NgModule} from '@angular/core';
import {NavbarComponent} from './navbar.component';
import {RouterModule} from '@angular/router';
import {SharedModule} from '../../shared/shared.module';

@NgModule({
    declarations: [NavbarComponent],
    imports: [
        SharedModule,
        RouterModule
    ],
    exports: [NavbarComponent]
})
export class NavbarModule {
}
