import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { WorkflowGraphModule} from 'workflow-graph';

import { AppComponent } from './app.component';
import { PreviewComponent } from './preview/preview.component';

@NgModule({
  declarations: [
    AppComponent,
    PreviewComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    NzIconModule,
    WorkflowGraphModule,
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule { }
