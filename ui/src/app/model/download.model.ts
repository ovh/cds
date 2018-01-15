export class Download {
    name: string;
    osArchs: Array<OSArch>;
}

export class OSArch {
    os: string;
    archs: Array<Arch>;
}

export class Arch {
    arch: string;
    available: boolean;
}
