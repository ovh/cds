export class PermissionValue {
    static READ = 4;
    static READ_EXECUTE = 5;
    static READ_WRITE_EXECUTE = 7;
}

export class Permission {
    readable: boolean;
    executable: boolean;
    writable: boolean;
}
