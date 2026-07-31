import { test, expect, beforeEach, afterEach } from 'vitest'
import {FileSystemSource} from "./source";
import {mkdtemp, rm, writeFile} from "node:fs/promises";
import {join} from "node:path";
import {tmpdir} from "node:os";

let dir: string
beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'fs-source-')) })
afterEach(async () => { await rm(dir, { recursive: true, force: true }) })

test('filesystem source reads from config file', async () => {
    const env = "test"
    const extension = "json"
    const fileContents = `
    {
        "project": "test-project",
        "instances": 2,
        "secret": "sm://this-is-secret:1",
        "flags": {
            "debugEndpoints": true,
            "noAuth": false
        }
    } 
    `

    await writeFile(join(dir, `${env}.${extension}`), fileContents)

    expect(await FileSystemSource(dir).getConfiguration(env, extension)).toBe(fileContents)
})

test('filesystem source fails if the requested file does not exist', async () => {
    const env = "test"
    const extension = "json"

    await expect(FileSystemSource(dir).getConfiguration(env, extension)).rejects.toThrow()
})

test('filesystem source fails if the requested directory does not exist', async () => {
    const env = "test"
    const extension = "json"

    await expect(FileSystemSource(`${dir}/unknown`).getConfiguration(env, extension)).rejects.toThrow()
})