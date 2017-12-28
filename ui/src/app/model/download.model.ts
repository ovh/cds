export class Download {
    name: string;
    osArchs: Array<OSArch>;
}

export class OSArch {
    os: string;
    archs: Array<string>;
}
