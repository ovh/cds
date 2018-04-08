+++
title = "Scopes & ACLs"
weight = 30

+++


{{<mermaid>}}
graph TD;
    A[fa:fa-suitcase Project contains] --> B(fa:fa-share-alt Workflows);
    A --> C(fa:fa-rocket Applications);
    A --> D(fa:fa-sitemap Pipelines);
    A --> E(fa:fa-tree Environments);
    click A "project/" "View CDS Project Documentation"
    click B "../concepts/workflow/" "View CDS Workflow Documentation"
    click C "application/" "View CDS Application Documentation"
    click D "../concepts/pipeline/" "View CDS Pipeline Documentation"
    click E "environment/" "View CDS Environment Documentation"
{{< /mermaid >}}

