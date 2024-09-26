import { HttpErrorResponse } from "@angular/common/http";

export class ErrorUtils {
    static print(e: any) {
        if (e instanceof HttpErrorResponse) {
            if (e.error) {
                return e.error.message;
            }
            return e.message;
        }
        return e;
    }
}
