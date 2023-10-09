import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, Output } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { FlatTreeControl, TreeControl } from '@angular/cdk/tree';
import { CollectionViewer, DataSource, SelectionChange } from '@angular/cdk/collections';
import { BehaviorSubject, merge, Observable } from 'rxjs';
import { first, map, tap } from 'rxjs/operators';
import { AnalysisEvent } from "../../service/analysis/analysis.service";
import {
    StatusAnalyzeError,
    StatusAnalyzeInProgress,
    StatusAnalyzeSkipped, StatusAnalyzeSucceed
} from "../../model/analysis.model";


// Represent a menu for a node
export interface MenuItem {
    name: string;
    route: string[];
    action: () => {};
}

export interface SelectedItem {
    id: string;
    name: string;
    type: string;
    child: SelectedItem;
    action: string;
}

// Represent the data tree inside the ngZorro component
export interface FlatNodeItem {
    expandable: boolean;
    clickable: boolean;
    expanded: boolean;
    id: string;
    name: string;
    parentNames: string[];
    type: string;
    icon?: string;
    iconTheme?: string;
    level: number;
    active: boolean;
    branch: string;
    loading?: boolean;
    menu: MenuItem[];
    select: FlatNodeItemSelect;
    onOpen: () => Observable<any>;
    loadChildren: () => Observable<FlatNodeItem[]>
}

export interface FlatNodeItemSelect {
    options: { key: string, value: string }[];
    selected: string;
    onchange: () => void;
}

export interface TreeEvent {
    eventType: string;
    node: FlatNodeItem;
}

class DynamicDatasource implements DataSource<FlatNodeItem> {
    private flattenedData: BehaviorSubject<FlatNodeItem[]>;
    private childrenLoadedSet = new Set<FlatNodeItem>();
    private mapRepoAnalyzeEvent = new Map<String, string[]>();
    private selectedNode: SelectedItem;

    constructor(private treeControl: TreeControl<FlatNodeItem>, initData: FlatNodeItem[]) {
        this.flattenedData = new BehaviorSubject<FlatNodeItem[]>(initData);
        treeControl.dataNodes = initData;
    }

    removeNode(id: string) {
        let currentNodes = this.flattenedData.getValue();
        let index = currentNodes.findIndex(n => n.id === id);
        if (index !== -1) {
            currentNodes.splice(index, 1);
            this.flattenedData.next(currentNodes);
        }
    }

    handleAnalysisEvent(event: AnalysisEvent): void {
        let nodes = this.flattenedData.getValue();
        let repoIndex = nodes.findIndex(n => n.id === event.repoID && n.type === 'repository');
        let repoNode = nodes[repoIndex];
        if (!repoNode) {
            return;
        }

        switch (event.status) {
            case StatusAnalyzeInProgress:
                if (!this.mapRepoAnalyzeEvent.has(repoNode.id)) {
                    this.mapRepoAnalyzeEvent.set(repoNode.id, []);
                }
                let analysesInProgress = this.mapRepoAnalyzeEvent.get(repoNode.id);
                if (!analysesInProgress || analysesInProgress.length === 0) {
                    analysesInProgress = [];
                } else {
                    let analyzeInProgressIndex = analysesInProgress.findIndex(id => id === event.analysisID);
                    if (analyzeInProgressIndex !== -1) {
                        return;
                    }
                }
                analysesInProgress.push(event.analysisID);
                this.mapRepoAnalyzeEvent.set(repoNode.id, analysesInProgress);
                repoNode.loading = true;
                break;
            case StatusAnalyzeSkipped:
            case StatusAnalyzeError:
            case StatusAnalyzeSucceed:
                if (!this.mapRepoAnalyzeEvent.has(repoNode.id)) {
                    return;
                }
                let analyses = this.mapRepoAnalyzeEvent.get(repoNode.id);
                let analyzeIndex = analyses.findIndex(id => id === event.analysisID);
                if (analyzeIndex === -1) {
                    return;
                }
                analyses.splice(analyzeIndex, 1);
                if (analyses.length === 0) {
                    repoNode.loading = false;
                }
                break;
        }
        this.flattenedData.next(nodes);
    }

    resetChildren(node: FlatNodeItem): void {
        if (this.childrenLoadedSet.has(node)) {
            this.childrenLoadedSet.delete(node);
            let nodes = this.flattenedData.getValue();
            let currentLevel = node.level;
            let currentIndex = nodes.findIndex(n => n.id === node.id);
            if (nodes.length > currentIndex + 1) {
                for (let i = currentIndex + 1; i < nodes.length; i++) {
                    let childNode = nodes[i];
                    if (childNode.level <= currentLevel) {
                        break;
                    }
                    nodes.splice(i, 1);
                    i--;
                }
                this.flattenedData.next(nodes);
            }
        }
        this.loadChildren(node).pipe(first()).subscribe(() => {
            this.selectNode(this.selectedNode);
        });
    }

    selectNode(node: SelectedItem) {
        if (!node) {
            return;
        }
        let currentNodes = this.flattenedData.getValue();
        if (currentNodes) {
            this.selectNodeRec(currentNodes, node, 0, []);
        }
        this.selectedNode = node;
    }

    selectNodeRec(currentNodes: FlatNodeItem[], node: SelectedItem, level: number, parents: string[]) {
        for (let i = 0; i < currentNodes.length; i++) {
            let n = currentNodes[i];
            if (n.level !== level) {
                continue;
            }
            if (n.id === node.id && n.type === node.type) {
                // Selected node found
                if (!node.child) {
                    currentNodes = currentNodes.map(no => {
                        no.active = false;
                        return no;
                    })
                    n.active = true;
                    this.flattenedData.next(currentNodes);
                    return;
                } else {
                    parents.push(n.name);
                    if (this.childrenLoadedSet.has(n)) {
                        if (node.child.action === 'select') {
                            let nodeIndex = currentNodes.findIndex(n => n.id === node.child.id)
                            if (nodeIndex === -1) {
                                currentNodes.splice(i + 1, 0, <FlatNodeItem>{
                                    id: node.child.id,
                                    name: node.child.name,
                                    parentNames: parents,
                                    level: level + 1,
                                    type: node.child.type,
                                    expandable: true
                                });
                                this.flattenedData.next(currentNodes);
                            }
                        }
                        this.selectNodeRec(this.flattenedData.getValue(), node.child, level + 1, parents);
                        this.treeControl.expand(n);
                    } else {
                        this.loadChildren(n).pipe(first()).subscribe(() => {
                            let nodes = this.flattenedData.getValue();
                            this.treeControl.expand(n);
                            this.selectNodeRec(nodes, node.child, level + 1, parents);
                        });
                    }
                }
            }
        }
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
            if (!node.expanded && treeControl.isExpanded(node) && node.onOpen) {
                node.onOpen().pipe(first()).subscribe(() => {
                    this.flattenedData.next(nodes);
                });
            }
            node.expanded = treeControl.isExpanded(node);
        });
        return results;
    }

    handleExpansionChange(change: SelectionChange<FlatNodeItem>): void {
        if (change.added) {
            change.added.forEach(node => this.loadChildren(node)?.pipe(first())?.subscribe());
        }
    }

    loadChildren(node: FlatNodeItem): Observable<any> {
        if (this.childrenLoadedSet.has(node) || !node.expandable) {
            return;
        }
        node.loading = true;
        return node.loadChildren().pipe(first(), map(children => {
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
                    flattenedData.splice(index + 1, 0, <FlatNodeItem>{
                        name: name,
                        type: 'info',
                        id: '',
                        level: node.level + 1,
                        expandable: false
                    });
                }
                this.childrenLoadedSet.add(node);
            }
            this.flattenedData.next(flattenedData);
        }));
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
            this.dataSource = new DynamicDatasource(this.treeControl, this._currentNodeTree);
        }
    }

    @Output() nodeEvent = new EventEmitter<TreeEvent>();

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    hasChild = (_: number, node: FlatNodeItem): boolean => node.expandable;

    clickOnNode(n: FlatNodeItem, e: Event): void {
        if (!n.expandable) {
            this.clickOnNodeLink(n, e);
            return;
        }
        if (!n.expanded) {
            this.treeControl.expand(n);
        } else {
            this.treeControl.collapse(n);
        }
        e.stopPropagation();
    }

    clickOnNodeLink(n: FlatNodeItem, e: Event): void {
        if (!n.clickable || n.active) { return; }
        this.nodeEvent.next({ node: n, eventType: 'select' });
        e.stopPropagation();
    }

    selectNode(s: SelectedItem): void {
        this.dataSource.selectNode(s);
        this._cd.markForCheck();
    }

    removeNode(id: string): void {
        this.dataSource.removeNode(id);
        this._cd.markForCheck();
    }

    handleAnalysisEvent(event: AnalysisEvent): void {
        this.dataSource.handleAnalysisEvent(event);
        this._cd.markForCheck();
    }

    refresh(): void {
        this._cd.markForCheck();
    }

    resetChildren(n: FlatNodeItem): void {
        this.dataSource.resetChildren(n);
        this._cd.markForCheck();
    }
}
