import {test, expect, afterEach, vi} from 'vitest'
import {Comfig} from "./comfig";
import {z} from "zod";
import {EnvResolver} from "./resolver";
import {expandable} from "./schema";
import {mkdtemp, rm, writeFile} from "node:fs/promises";
import {tmpdir} from "node:os";
import {join} from "node:path";

const getFunctionalComfig = <T extends z.ZodObject>(input: unknown, schema: T) => {
    return new Comfig(schema)
        .useSource({
            getConfiguration: (environment: string, extension: string) => JSON.stringify(input)
        })
}

afterEach(() => { vi.unstubAllEnvs() })

test('loading configuration throws when reading the source fails', async () => {
    const comfig = getFunctionalComfig({}, z.object({ test: z.string() }))

    comfig.useSource({
        getConfiguration: (environment: string, extension: string) => {
            throw Error("source fails")
        }
    })

    await expect(comfig.load()).rejects.toThrow()
})

test('loading configuration throws when parsing the source fails', async () => {
    const comfig = getFunctionalComfig({ test: "this is valid" }, z.object({ test: z.string() }))

    comfig.useParser({
        extension: "json",
        parse: (raw: string) => {
            throw Error("parser fails")
        }
    })

    await expect(comfig.load()).rejects.toThrow()
})

test('loading configuration throws when creating a resolver fails', async () => {
    const comfig = getFunctionalComfig({ test: "this is valid" }, z.object({ test: z.string() }))

    comfig.useResolver((config) => {
        throw Error("resolver fails")
    })

    await expect(comfig.load()).rejects.toThrow("resolver fails")
})

test('loading configuration throws when duplicate resolver prefixes are found', async () => {
    const comfig = getFunctionalComfig({ test: "this is valid" }, z.object({ test: z.string() }))

    comfig.useResolver((config) => EnvResolver())
    comfig.useResolver((config) => EnvResolver())

    await expect(comfig.load()).rejects.toThrow()
})

test('should use correct prefix for resolver', async () => {
    const comfig = getFunctionalComfig({ test: "custom://env://test://something://hello" }, z.object({ test: z.string() }))

    let wasAccessed = false
    let rest = ""
    comfig.useResolver((config) => EnvResolver())
    comfig.useResolver((config) => ({
        prefix: "custom",
        resolve: (value: string) => {
            wasAccessed = true
            rest = value
            return "success"
        }
    }))

    const output = await comfig.load()

    expect(rest).eq("env://test://something://hello")
    expect(wasAccessed).true
    expect(output.test).eq("success")
})

test('no reducer found for prefix should leave value as is', async () => {
    const comfig = getFunctionalComfig({ test: "https://example.com" }, z.object({ test: z.string() }))

    comfig.useResolver((config) => EnvResolver())

    expect((await comfig.load()).test).eq("https://example.com")
})

test('prefix has to match exactly to the start of the string', async () => {
    const comfig = getFunctionalComfig({ test: "eenv://test" }, z.object({ test: z.string() }))

    comfig.useResolver((config) => EnvResolver())

    expect((await comfig.load()).test).eq("eenv://test") //does not get env variable since eenv != env
})

test('loading configuration reports the nested path when resolving a value fails', async () => {
    const comfig = getFunctionalComfig(
        { database: { password: "secret://primary" } },
        z.object({ database: z.object({ password: z.string() }) }),
    )
    comfig.useResolver(() => ({
        prefix: "secret",
        resolve: () => { throw Error("access denied") },
    }))

    await expect(comfig.load()).rejects.toThrow(
        'failed to resolve "secret://primary" at $.database.password',
    )
})

test('loading configuration resolves values inside arrays', async () => {
    const comfig = getFunctionalComfig(
        { secrets: ["secret://first", "secret://second"] },
        z.object({ secrets: z.array(z.string()) }),
    )
    comfig.useResolver(async () => ({
        prefix: "secret",
        resolve: async (value) => `resolved-${value}`,
    }))

    await expect(comfig.load()).resolves.toEqual({
        secrets: ["resolved-first", "resolved-second"],
    })
})

test('loading configuration validates the resolved configuration', async () => {
    const comfig = getFunctionalComfig(
        { port: "not-a-number" },
        z.object({ port: z.number() }),
    )

    await expect(comfig.load()).rejects.toBeInstanceOf(z.ZodError)
})

test('usePath loads the correct file using the parser extension', async () => {
    const directory = await mkdtemp(join(tmpdir(), "comfig-"))
    vi.stubEnv("env", "integration")
    vi.resetModules()
    const {Comfig: EnvironmentComfig} = await import("./comfig")
    await writeFile(join(directory, "integration.conf"), "enabled")

    try {
        const comfig = new EnvironmentComfig(z.object({ enabled: z.boolean() }))
            .usePath(directory)
            .useParser({
                extension: "conf",
                parse: (raw) => ({ enabled: raw === "enabled" }),
            })

        await expect(comfig.load()).resolves.toEqual({ enabled: true })
    } finally {
        await rm(directory, { recursive: true, force: true })
    }
})

test('loading configuration happy path should work', async () => {
    const resolved = "value"
    vi.stubEnv("resolved", resolved)
    vi.stubEnv("resolvedStructure", "John|50")

    const raw = {
        resolved: "env://resolved",
        resolvedStructure: "env://resolvedStructure",
        notResolved: "eenv://resolved",
        numberValue: 2.5,
        flags: {
            noAuth: false
        }
    }

    const comfig = getFunctionalComfig(raw, z.object({
        resolved: z.string(),
        resolvedStructure: expandable(z.object({
            name: z.string(),
            age: z.number()
        }), (data) => {
            const parts = data.split("|")

            return {
                name: parts[0],
                age: parseInt(parts[1])
            }
        }),
        notResolved: z.string(),
        numberValue: z.number(),
        flags: z.object({
            noAuth: z.boolean()
        })
    }))

    comfig.useResolver((config) => EnvResolver())

    const configuration = await comfig.load()

    expect(configuration.resolved).eq(resolved)
    expect(configuration.resolvedStructure).toEqual({ name: "John", age: 50 })
    expect(configuration.notResolved).toEqual(raw.notResolved)
    expect(configuration.numberValue).eq(raw.numberValue)
    expect(configuration.flags.noAuth).false
})