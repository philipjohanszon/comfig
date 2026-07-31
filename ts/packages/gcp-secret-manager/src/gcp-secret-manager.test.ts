import {expect, test, vi, type Mock} from "vitest"
import type {SecretManagerServiceClient} from "@google-cloud/secret-manager"
import {GCPSecretResolver, type GCPSecretResolverOptions} from "./gcp-secret-manager"

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

    await expect(resolver.resolve("db")).rejects.toThrow()
})

test('gcp resolver bare name resolves the latest version', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("s3cr3t"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    expect(await resolver.resolve("db")).eq("s3cr3t")
    expect(accessSecretVersion).toHaveBeenCalledWith({name: "projects/my-proj/secrets/db/versions/latest"})
})

test('gcp resolver @latest resolves the latest version', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("s3cr3t"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    expect(await resolver.resolve("db@latest")).eq("s3cr3t")
    expect(accessSecretVersion).toHaveBeenCalledWith({name: "projects/my-proj/secrets/db/versions/latest"})
})

test('gcp resolver @version fetches that version', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("old"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    expect(await resolver.resolve("db@3")).eq("old")
    expect(accessSecretVersion).toHaveBeenCalledWith({name: "projects/my-proj/secrets/db/versions/3"})
})

test('gcp resolver rejects an empty version selector', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("secret"))
    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await expect(resolver.resolve("db@")).rejects.toThrow()
    expect(accessSecretVersion).not.toHaveBeenCalled()
})

test('gcp resolver rejects multiple version selectors', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("secret"))
    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await expect(resolver.resolve("db@3@typo")).rejects.toThrow()
    expect(accessSecretVersion).not.toHaveBeenCalled()
})

test('gcp resolver builds resource name from projectId and short name', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue(payload("v"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await resolver.resolve("api-key")

    expect(accessSecretVersion).toHaveBeenCalledWith({name: "projects/my-proj/secrets/api-key/versions/latest"})
})

test('gcp resolver throws when secret doesn\'t exist', async () => {
    const accessSecretVersion = vi.fn().mockRejectedValue(new Error("NOT_FOUND: Secret [db] not found"))

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await expect(resolver.resolve("db")).rejects.toThrow()
})

test('gcp resolver throws when secret has no payload', async () => {
    const accessSecretVersion = vi.fn().mockResolvedValue([{payload: {data: null}}])

    const resolver = GCPSecretResolver({projectId: "my-proj", client: clientWith(accessSecretVersion)})

    await expect(resolver.resolve("db")).rejects.toThrow()
})
