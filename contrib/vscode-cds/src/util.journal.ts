import { OutputChannel, window } from "vscode";
import { Property } from "./util.property";

export enum LogCategory {
    Info = "info",
    Error = "error",
    Critical = "critical",
}

export class Journal {
    public static logInfo(message: string) {
        Journal.getInstance().writeToLog(LogCategory.Info, message);
    }

    public static logError(error: Error): void {
        Journal.getInstance().writeToLog(
            LogCategory.Error,
            error.toString(),
        );
    }

    public static logCritical(error: Error, message: string): void {
        Journal.getInstance().writeToLog(
            LogCategory.Critical,
            error.toString(),
        );
        Journal.getInstance().showErrorMessage(message);
    }

    public static getInstance(): Journal {
        if (!Journal.instance) {
            Journal.instance = new Journal();
        }

        return Journal.instance;
    }

    private static instance: Journal;

    private static timestamp(): string {
        const now = new Date();
        const hour = now
            .getHours()
            .toString()
            .padStart(2, "0");
        const minute = now
            .getMinutes()
            .toString()
            .padStart(2, "0");
        const second = now
            .getSeconds()
            .toString()
            .padStart(2, "0");

        return `${hour}:${minute}:${second}`;
    }

    private readonly outputChannel: OutputChannel;

    private constructor() {
        this.outputChannel = window.createOutputChannel("CDS");
    }

    public dispose() {
        this.outputChannel.dispose();
    }

    private async showErrorMessage(message: string): Promise<void> {
        const titleShowLog = "Show Log";

        const selectedItem = await window.showErrorMessage(
            message,
            titleShowLog,
        );

        if (selectedItem === titleShowLog) {
            this.outputChannel.show();
        }
    }

    private writeToLog(category: LogCategory, message: string): boolean {
        const allowCategory = this.logCategoryAllowed(category);

        if (allowCategory) {
            const trimmedMessage = message.trim();
            const timestamp = Journal.timestamp();
            this.outputChannel.appendLine(
                `[ ${timestamp} | ${category} ] ${trimmedMessage}`,
            );
        }

        return allowCategory;
    }

    private logCategoryAllowed(level: LogCategory): boolean {
        const enabledLevels = Property.get("logLevel");

        if (enabledLevels) {
            return enabledLevels.includes(level);
        }
        return false;
    }
}
