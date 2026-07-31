import {expect, test, vi, type Mock} from "vitest"
import type {SecretsManagerClient} from "@aws-sdk/client-secrets-manager"
import {AWSSecretsResolver} from "./aws-secrets-manager"

const clientWith = (send: Mock): SecretsManagerClient =>
    ({send}) as unknown as SecretsManagerClient

test('aws resolver bare name resolves the current version', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "s3cr3t"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod/db/password")).eq("s3cr3t")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password"}}))
})

test('aws resolver rejects empty secret name', async () => {
    const send = vi.fn()

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("")).rejects.toThrow()
    expect(send).not.toHaveBeenCalled()
})

test('aws resolver @label fetches that staging label', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "prev"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod/db/password@AWSPREVIOUS")).eq("prev")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password", VersionStage: "AWSPREVIOUS"}}))
})

test('aws resolver @latest fetches the latest staging label', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "latest"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod/db/password@latest")).eq("latest")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password", VersionStage: "latest"}}))
})

test('aws resolver #versionId fetches by VersionId', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "pinned"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    const id = "abcdef12-3456-7890-abcd-ef1234567890"
    expect(await resolver.resolve(`prod/db/password#${id}`)).eq("pinned")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password", VersionId: id}}))
})

test('aws resolver @stage decodes an escaped @ in a secret name', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "previous"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod%40blue/db/password@AWSPREVIOUS")).eq("previous")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({
        input: {SecretId: "prod@blue/db/password", VersionStage: "AWSPREVIOUS"}
    }))
})

test('aws resolver rejects an unescaped @ in a secret name with a stage', async () => {
    const send = vi.fn()

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("prod@blue/db/password@AWSPREVIOUS")).rejects.toThrow()
    expect(send).not.toHaveBeenCalled()
})

test('aws resolver decodes an escaped @ in a secret name', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "current"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod%40blue/db/password")).eq("current")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod@blue/db/password"}}))
})

test('aws resolver @stage fetches a custom staging label', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "canary"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod/db/password@canary-2026")).eq("canary")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({
        input: {SecretId: "prod/db/password", VersionStage: "canary-2026"}
    }))
})

test('aws resolver #versionId does not infer the version ID format', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "pinned"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})
    const id = "version-id-which-is-not-a-canonical-uuid"

    expect(await resolver.resolve(`prod/db/password#${id}`)).eq("pinned")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password", VersionId: id}}))
})

test('aws resolver rejects an empty version stage before calling AWS', async () => {
    const send = vi.fn()

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("prod/db/password@")).rejects.toThrow()
    expect(send).not.toHaveBeenCalled()
})

test('aws resolver rejects malformed percent encoding before calling AWS', async () => {
    const send = vi.fn()

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("prod%4/db/password")).rejects.toThrow()
    expect(send).not.toHaveBeenCalled()
})

test('aws resolver decodes SecretBinary', async () => {
    const send = vi.fn().mockResolvedValue({SecretBinary: Buffer.from("binary-secret", "utf8")})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod/db/password")).eq("binary-secret")
})

test('aws resolver throws when secret doesn\'t exist', async () => {
    const send = vi.fn().mockRejectedValue(new Error("not found"))

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("prod/db/password")).rejects.toThrow()
})

test('aws resolver propagates AWS authorization errors', async () => {
    const error = new Error("AccessDeniedException")
    const send = vi.fn().mockRejectedValue(error)

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("prod/db/password")).rejects.toBe(error)
})

test('aws resolver throws when secret has no value', async () => {
    const send = vi.fn().mockResolvedValue({})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("prod/db/password")).rejects.toThrow()
})
