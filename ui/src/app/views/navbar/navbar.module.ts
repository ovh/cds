import {NgModule} from '@angular/core';
import {RouterModule} from '@angular/router';
import {SharedModule} from '../../shared/shared.module';
import {NavbarComponent} from './navbar.component';

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
