import {expect, test, afterEach, vi} from 'vitest'
import {EnvResolver, FileResolver} from "./resolver";
import {mkdtemp, writeFile, rm} from "node:fs/promises";
import {tmpdir} from "node:os";
import {join} from "node:path";

afterEach(() => { vi.unstubAllEnvs() })

test('env resolver finds environment variable', () => {
    vi.stubEnv("test", "hello")

    const resolver = EnvResolver()

    //would appear as env://test unless prefix is overridden
    expect(resolver.resolve("test")).eq("hello")
})

test('env resolver throws if environment variable is not found', () => {
    delete process.env.test

    const resolver = EnvResolver()

    expect(() => resolver.resolve("test")).to.throw()
})

test('file resolver reads file contents', async () => {
    const dir = await mkdtemp(join(tmpdir(), "comfig-"))
    const file = join(dir, "secret")
    await writeFile(file, "hello from file")

    const resolver = FileResolver()

    //would appear as file://<path> unless prefix is overridden
    expect(await resolver.resolve(file)).eq("hello from file")

    await rm(dir, { recursive: true, force: true })
})

test('file resolver throws if directory is not found', async () => {
    const resolver = FileResolver()
    await expect(resolver.resolve("/comfig/does/not/exist")).rejects.toThrow()
})

test('file resolver throws if file is not found', async () => {
    const dir = await mkdtemp(join(tmpdir(), "comfig-"))
    const file = join(dir, "secret")
    await writeFile(file, "hello from file")

    const resolver = FileResolver()

    await expect(resolver.resolve(join(dir, "not-secret"))).rejects.toThrow()

    await rm(dir, { recursive: true, force: true })
})