import { NgModule } from '@angular/core';
import { GraphComponent } from './graph.component';
import { GraphStageNodeComponent } from './node/stage-node.component';
import { GraphForkJoinNodeComponent } from './node/fork-join-node.components';
import { GraphJobNodeComponent } from './node/job-node.component';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzAvatarModule } from 'ng-zorro-antd/avatar';
import { NzToolTipModule } from 'ng-zorro-antd/tooltip';
import { CommonModule } from '@angular/common';
import { NzButtonModule } from 'ng-zorro-antd/button';
import {NzBadgeModule } from 'ng-zorro-antd/badge';
import { AimOutline, RotateRightOutline, RotateLeftOutline, PlayCircleOutline, WarningOutline } from '@ant-design/icons-angular/icons';
import { IconDefinition } from '@ant-design/icons-angular';
import { GraphMatrixNodeComponent } from './node/matrix-node.component';
import { IsJobTerminatedPipe } from './is-job-terminated.pipe';
import { NzPopconfirmModule } from 'ng-zorro-antd/popconfirm';

const icons: IconDefinition[] = [AimOutline, RotateRightOutline, RotateLeftOutline, PlayCircleOutline, WarningOutline];

@NgModule({
  declarations: [
    GraphComponent,
    GraphForkJoinNodeComponent,
    GraphJobNodeComponent,
    GraphMatrixNodeComponent,
    GraphStageNodeComponent,
    IsJobTerminatedPipe
  ],
  imports: [
    CommonModule,
    NzAvatarModule,
    NzBadgeModule,
    NzButtonModule,
    NzIconModule.forRoot(icons),
    NzPopconfirmModule,
    NzToolTipModule
  ],
  exports: [
    GraphComponent,
    GraphForkJoinNodeComponent,
    GraphJobNodeComponent,
    GraphMatrixNodeComponent,
    GraphStageNodeComponent,
    IsJobTerminatedPipe
  ]
})
export class GraphModule { }