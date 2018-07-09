export class Broadcast {
    id: number;
    title: string;
    content: string;
    level: string;
    created: string;
    updated: string;
    archived: boolean;
    project_key: string;
    read: boolean;

    updating: boolean;

    static fromEvent(bEvent: BroadcastEvent): Broadcast {
        let b = new Broadcast();
        b.id = bEvent.ID;
        b.read = bEvent.Read;
        b.archived = bEvent.Archived;
        b.content = bEvent.Content;
        b.created = bEvent.Created;
        b.level = bEvent.Level;
        b.project_key = bEvent.ProjectKey;
        b.title = bEvent.Title;
        b.updated = bEvent.Updated;
        return b;
    }
}

export class BroadcastEvent {
    ID: number;
    Title: string;
    Content: string;
    Level: string;
    Created: string;
    Updated: string;
    Archived: boolean;
    ProjectKey: string;
    Read: boolean;
}
