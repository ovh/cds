import { exec } from "child_process";
import { dirname } from "path";

import { Journal } from "./journal";

export async function getGitRepositoryPath(filename: string): Promise<string> {
    return (await gitExec(dirname(filename), "rev-parse", "--show-toplevel")).trim();
}

export function setGitLocalConfig(repositoryPath: string, section: string, key: string, value: string) {
    return gitExec(repositoryPath, "config", "--local", "--replace-all", `${section}.${key}`, value);
}

export async function getGitLocalConfig(repositoryPath: string, section: string, key: string): Promise<string | null> {
    const result = await gitExec(repositoryPath, "config", "--local", "--get", `${section}.${key}`);

    if (result) {
        return result.trim();
    }

    return null;
}

function gitExec(repositoryPath: string, ...args: string[]): Promise<string> {
    const cmd = `git ${args.join(" ")}`;

    Journal.logInfo(`running command ${cmd} from directory ${repositoryPath}`);

    return new Promise((resolve, reject) => {
        exec(cmd,
            {
                cwd: repositoryPath,
            },
            (error, stdout, stderr) => {
                Journal.logInfo(stdout)
                Journal.logInfo(stderr)


                if (error) {
                    Journal.logError(error);
                    reject(error);
                }
                if (stderr) {
                    reject(stderr);
                }
                resolve(stdout);
            });
    });
}
