import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { NzTreeFlatDataSource, NzTreeFlattener } from 'ng-zorro-antd/tree-view';
import { FlatTreeControl } from '@angular/cdk/tree';

// Represent the data tree
export interface NodeItem {
    name: string;
    icon?: string;
    iconTheme?: string;
    children?: NodeItem[];
    menu?: MenuItem[];
}

// Represent a menu for a node
export interface MenuItem {
    name: string;
    route: string;
}

// Represent the data tree inside the ngZorro component
interface FlatNodeItem {
    expandable: boolean;
    name: string;
    icon: string;
    iconTheme: string;
    level: number;
    menu: MenuItem[];
}

@Component({
    selector: 'app-tree',
    templateUrl: './tree.html',
    styleUrls: ['./tree.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class TreeComponent {
    // Transform a node into flatnode
    transformer = (node: NodeItem, level: number): FlatNodeItem => ({
        expandable: !!node.children && node.children.length > 0,
        name: node.name,
        icon: node.icon,
        iconTheme: node.iconTheme,
        menu: node.menu,
        level
    });
    treeControl = new FlatTreeControl<FlatNodeItem>(
        node => node.level,
        node => node.expandable
    );
    treeFlattener = new NzTreeFlattener(
        this.transformer,
        node => node.level,
        node => node.expandable,
        node => node.children
    );
    dataSource = new NzTreeFlatDataSource(this.treeControl, this.treeFlattener);

    _currentNodeTree: NodeItem[];
    get tree(): NodeItem[] {
        return this._currentNodeTree;
    }
    @Input() set tree(data: NodeItem[]) {
        this._currentNodeTree = data;
        this.dataSource.setData(this._currentNodeTree);
    }

    hasChild = (_: number, node: FlatNodeItem): boolean => node.expandable;

}
