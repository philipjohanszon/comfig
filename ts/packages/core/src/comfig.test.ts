import {test, expect, afterEach, vi} from 'vitest'
import {Comfig} from "./comfig";
import {z} from "zod";
import {EnvResolver} from "./resolver";
import {expandable} from "./schema";

const getFunctionalComfig = <T extends z.ZodObject>(input: unknown, schema: T) => {
    return new Comfig(schema)
        .useSource({
            getConfiguration: (environment: string, extension: string) => JSON.stringify(input)
        })
}

afterEach(() => { vi.unstubAllEnvs() })

test('loading configuration should throw if no path or source is setup', () => {
    const comfig = new Comfig(z.object({ test: z.string() }))

    expect(() => comfig.load()).rejects.toThrow()
})

test('loading configuration should throw if reading source fails', () => {
    const comfig = getFunctionalComfig({}, z.object({ test: z.string() }))

    comfig.useSource({
        getConfiguration: (environment: string, extension: string) => {
            throw Error("source fails")
        }
    })

    expect(() => comfig.load()).rejects.toThrow()
})

test('loading configuration should throw if parsing source fails', () => {
    const comfig = getFunctionalComfig({ test: "this is valid" }, z.object({ test: z.string() }))

    comfig.useParser({
        extension: "json",
        parse: (raw: string) => {
            throw Error("parser fails")
        }
    })

    expect(() => comfig.load()).rejects.toThrow()
})

test('loading configuration should throw if resolver fails', () => {
    const comfig = getFunctionalComfig({ test: "this is valid" }, z.object({ test: z.string() }))

    comfig.useResolver((config) => {
        throw Error("resolver fails")
    })

    expect(() => comfig.load()).rejects.toThrow()
})

test('loading configuration should throw if duplicate resolver prefixes are encountered', () => {
    const comfig = getFunctionalComfig({ test: "this is valid" }, z.object({ test: z.string() }))

    comfig.useResolver((config) => EnvResolver())
    comfig.useResolver((config) => EnvResolver())

    expect(() => comfig.load()).rejects.toThrow()
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