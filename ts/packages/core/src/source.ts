import {readFile} from "node:fs/promises";

export interface Source {
    getConfiguration: (environment: string, extension: string) => Promise<string> | string
}

export const FileSystemSource = (path: string) => ({
    getConfiguration: async (environment: string, extension: string) => {
        const dir = path.endsWith("/") ? path.slice(0, -1) : path
        return await readFile(`${dir}/${environment}.${extension}`, "utf8")
    }
} satisfies Source)