import {expect, test, vi, type Mock} from "vitest"
import type {SecretManagerServiceClient} from "@google-cloud/secret-manager"
import {GCPSecretResolver, GCPSecretResolverOptions} from "./gcp-secret-manager"

const clientWith = (accessSecretVersion: Mock): SecretManagerServiceClient =>
    ({accessSecretVersion}) as unknown as SecretManagerServiceClient

const payload = (value: string) => [{payload: {data: Buffer.from(value, "utf8")}}]

test('gcp resolver throws when constructed without projectId', () => {
    expect(() => GCPSecretResolver({projectId: ""})).to.throw()
    expect(() => GCPSecretResolver({} as unknown as GCPSecretResolverOptions)).to.throw()
})

test('gcp resolver throws if client can\'t authenticate', async () => {
    const accessSecretVersion = vi.fn().mockRejectedValue(new Error("UNAUTHENTICATED: could not load the default credentials"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await expect(resolver.resolve("db@latest")).rejects.toThrow()
})

test('gcp resolver throws error if key@[version/latest] format is not followed', async () => {
    const accessSecretVersion = vi.fn()

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    //no @version suffix
    await expect(resolver.resolve("db")).rejects.toThrow()
    expect(accessSecretVersion).not.toHaveBeenCalled()
})

test('gcp resolver key@version format fetches secret at version', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("s3cr3t"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    expect(await resolver.resolve("db@3")).eq("s3cr3t")
    expect(accessSecretVersion).toHaveBeenCalledWith({name: "projects/my-proj/secrets/db/versions/3"})
})

test('gcp resolver key@latest format fetches secret at latest version', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("s3cr3t"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    expect(await resolver.resolve("db@latest")).eq("s3cr3t")
    expect(accessSecretVersion).toHaveBeenCalledWith({name: "projects/my-proj/secrets/db/versions/latest"})
})

test('gcp resolver builds resource name from projectId and short name', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("v"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await resolver.resolve("api-key@latest")

    expect(accessSecretVersion).toHaveBeenCalledWith({name: "projects/my-proj/secrets/api-key/versions/latest"})
})

test('gcp resolver throws when secret doesn\'t exist', async () => {
    const accessSecretVersion = vi.fn().mockRejectedValue(new Error("not found"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await expect(resolver.resolve("db@latest")).rejects.toThrow()
})

test('gcp resolver throws when secret has no payload', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue([{payload: {data: null}}])

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await expect(resolver.resolve("db@latest")).rejects.toThrow()
})
