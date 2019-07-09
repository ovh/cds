import { ChangeDetectionStrategy, Component, Input } from '@angular/core';

export class PathItem {
    icon: string;
    translate: string;
    text: string;
    active: boolean;
    routerLink: Array<string>;
    queryParams: any;
}

@Component({
    selector: 'app-breadcrumb',
    templateUrl: './breadcrumb.html',
    styleUrls: ['./breadcrumb.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class BreadcrumbComponent {
    @Input() path: Array<PathItem>;
}
