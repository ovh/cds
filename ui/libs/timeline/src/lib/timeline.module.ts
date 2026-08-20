import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { OverlayModule } from '@angular/cdk/overlay';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzSpaceModule } from 'ng-zorro-antd/space';
import { NzTooltipModule } from 'ng-zorro-antd/tooltip';
import { TimelineComponent } from './timeline.component';
import { TIMELINE_ICONS } from './timeline.icons';

@NgModule({
    declarations: [
        TimelineComponent
    ],
    imports: [
        CommonModule,
        OverlayModule,
        NzButtonModule,
        NzIconModule.forRoot(TIMELINE_ICONS),
        NzSpaceModule,
        NzTooltipModule
    ],
    exports: [
        TimelineComponent
    ]
})
export class TimelineModule { }
