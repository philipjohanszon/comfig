import {expect, test, afterEach, vi} from 'vitest'
import {EnvResolver} from "./resolver";

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