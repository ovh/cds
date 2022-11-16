import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    OnDestroy,
    ViewChild
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {NodeItem} from 'app/shared/tree/tree.component';
import { NzCollapseComponent, NzCollapsePanelComponent } from 'ng-zorro-antd/collapse';

@Component({
    selector: 'app-projectv2-sidebar',
    templateUrl: './project.sidebar.html',
    styleUrls: ['./project.sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2SidebarComponent implements OnDestroy {
    currentWorkspace: NodeItem[];
    currentIntegrations: NodeItem[];
    panels: boolean[] = [true, true, false]

    ngOnDestroy(): void {
    }

    constructor(private _cd: ChangeDetectorRef) {
        this.currentWorkspace = [];
        this.currentWorkspace.push({ name: "Github", menu: [{name: "Add a repository", route: "/add/repo"}],
                children: [
                    {name: "sguiheux/repo1", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                    {name: "sguiheux/repo2", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                ]},
            { name: "Bitbucket", menu: [{name: "Add a repository", route: "/add/repo"}], children: [
                    {name: "sguiheux/repo1", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                    {name: "sguiheux/repo2", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                ]},
            { name: "Bitbucket", menu: [{name: "Add a repository", route: "/add/repo"}], children: [
                    {name: "sguiheux/repo1", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                    {name: "sguiheux/repo2", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                ]},
            { name: "Bitbucket", children: [
                    {name: "sguiheux/repo1", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                    {name: "sguiheux/repo2", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                ]},
            { name: "Bitbucket", children: [
                    {name: "sguiheux/repo1", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                    {name: "sguiheux/repo2", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                ]},
            { name: "Bitbucket", children: [
                    {name: "sguiheux/repo1", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                    {name: "sguiheux/repo2", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                ]},
            { name: "Bitbucket", children: [
                    {name: "sguiheux/repo1", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                    {name: "sguiheux/repo2", children: [
                            {name:"Workflows", children: [
                                    {name: "workflow 1", icon: "file", iconTheme: "outline"},{name: "workflow 2", icon: "file", iconTheme: "outline"},
                                ]
                            }
                        ]
                    },
                ]});

        this.currentIntegrations = [];
        this.currentIntegrations.push({ name: "Arsenal",
                children: [
                    {name: "eu"},
                    {name: "labeu"},
                ]},
            { name: "Artifactory", children: [
                    {name: "digital-core-platform-cicd"},
                    {name: "digital-core-platform-pass"},
                ]});
    }

    togglePanel(i: number): void {
        this.panels[i] = !this.panels[i];
        this.panels = Object.assign([], this.panels)
        this._cd.markForCheck();
    }
}
