import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { FlatTreeControl, TreeControl } from '@angular/cdk/tree';
import { CollectionViewer, DataSource, SelectionChange } from '@angular/cdk/collections';
import { BehaviorSubject, merge, Observable, of } from 'rxjs';
import { first, map, tap } from 'rxjs/operators';


// Represent a menu for a node
export interface MenuItem {
    name: string;
    route: string[];
}

// Represent the data tree inside the ngZorro component
export interface FlatNodeItem {
    expandable: boolean;
    id: string;
    name: string;
    type: string;
    icon?: string;
    iconTheme?: string;
    level: number;
    loading?: boolean
    menu: MenuItem[];
    loadChildren: () => Observable<FlatNodeItem[]>
}

export interface TreeEvent {
    eventType: string;
    node: FlatNodeItem;
}

class DynamicDatasource implements DataSource<FlatNodeItem> {
    private flattenedData: BehaviorSubject<FlatNodeItem[]>;
    private childrenLoadedSet = new Set<FlatNodeItem>();

    constructor(private treeControl: TreeControl<FlatNodeItem>, initData: FlatNodeItem[]) {
        this.flattenedData = new BehaviorSubject<FlatNodeItem[]>(initData);
        treeControl.dataNodes = initData;
    }

    connect(collectionViewer: CollectionViewer): Observable<FlatNodeItem[]> {
        const changes = [
            collectionViewer.viewChange,
            this.treeControl.expansionModel.changed.pipe(tap(change => this.handleExpansionChange(change))),
            this.flattenedData
        ];
        return merge(...changes).pipe(map(() => this.expandFlattenedNodes(this.flattenedData.getValue())));
    }

    expandFlattenedNodes(nodes: FlatNodeItem[]): FlatNodeItem[] {
        const treeControl = this.treeControl;
        const results: FlatNodeItem[] = [];
        const currentExpand: boolean[] = [];
        currentExpand[0] = true;

        nodes.forEach(node => {
            let expand = true;
            for (let i = 0; i <= treeControl.getLevel(node); i++) {
                expand = expand && currentExpand[i];
            }
            if (expand) {
                results.push(node);
            }
            if (treeControl.isExpandable(node)) {
                currentExpand[treeControl.getLevel(node) + 1] = treeControl.isExpanded(node);
            }
        });
        return results;
    }

    handleExpansionChange(change: SelectionChange<FlatNodeItem>): void {
        if (change.added) {
            change.added.forEach(node => this.loadChildren(node));
        }
    }

    loadChildren(node: FlatNodeItem): void {
        if (this.childrenLoadedSet.has(node) || !node.expandable) {
            return;
        }
        node.loading = true;
        node.loadChildren().pipe(first()).subscribe(children => {
            node.loading = false;
            const flattenedData = this.flattenedData.getValue();
            const index = flattenedData.indexOf(node);
            if (index !== -1) {
                if (children.length > 0) {
                    flattenedData.splice(index + 1, 0, ...children);
                } else {
                    let name = '';
                    switch (node.type) {
                        case 'vcs':
                            name = 'There is no repository';
                            break;
                        case 'repository':
                            name = 'There is no cds files';
                            break;
                    }
                    flattenedData.splice(index + 1, 0, <FlatNodeItem>{name: name, type: 'info', id: '', level: node.level+1, expandable: false});
                }
                this.childrenLoadedSet.add(node);
            }
            this.flattenedData.next(flattenedData);
        });
    }

    disconnect(): void {
        this.flattenedData.complete();
    }
}

@Component({
    selector: 'app-tree',
    templateUrl: './tree.html',
    styleUrls: ['./tree.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class TreeComponent {
    treeControl = new FlatTreeControl<FlatNodeItem>(
        node => node.level,
        node => node.expandable
    );

    dataSource: DynamicDatasource;

    _currentNodeTree: FlatNodeItem[];
    get tree(): FlatNodeItem[] {
        return this._currentNodeTree;
    }
    @Input() set tree(data: FlatNodeItem[]) {
        this._currentNodeTree = data;
        if (data) {
            this.dataSource = new DynamicDatasource(this.treeControl,  this._currentNodeTree);
        }
    }

    @Output() nodeEvent = new EventEmitter<TreeEvent>();

    hasChild = (_: number, node: FlatNodeItem): boolean => node.expandable;

    clickOnNode(t: string, n: FlatNodeItem): void {
        this.nodeEvent.next({node: n, eventType: t})
    }
}
