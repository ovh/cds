export interface Project {
    readonly key: string;
    readonly name: string;
    readonly description: string
    readonly favorite: 'true' | 'false';
    readonly found: boolean;
}
