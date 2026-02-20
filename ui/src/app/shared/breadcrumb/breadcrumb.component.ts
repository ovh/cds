import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { ThemeType } from '@ant-design/icons-angular';

export class PathItem {
    icon: string;
    iconTheme: ThemeType;
    translate: string;
    text: string;
    active: boolean;
    routerLink: Array<string>;
    queryParams: any;
}

@Component({
    standalone: false,
    selector: 'app-breadcrumb',
    templateUrl: './breadcrumb.html',
    styleUrls: ['./breadcrumb.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class BreadcrumbComponent {
    @Input() path: Array<PathItem>;
}
