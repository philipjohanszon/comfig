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

test('aws resolver @latest resolves the current version (no VersionStage)', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "s3cr3t"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod/db/password@latest")).eq("s3cr3t")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password"}}))
})

test('aws resolver @label fetches that staging label', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "prev"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("prod/db/password@AWSPREVIOUS")).eq("prev")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password", VersionStage: "AWSPREVIOUS"}}))
})

test('aws resolver @uuid fetches by VersionId', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "pinned"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    const id = "abcdef12-3456-7890-abcd-ef1234567890"
    expect(await resolver.resolve(`prod/db/password@${id}`)).eq("pinned")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password", VersionId: id}}))
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

test('aws resolver throws when secret has no value', async () => {
    const send = vi.fn().mockResolvedValue({})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("prod/db/password")).rejects.toThrow()
})
