import {expect, test, vi, type Mock} from "vitest"
import type {SecretsManagerClient} from "@aws-sdk/client-secrets-manager"
import {AWSSecretsResolver} from "./aws-secrets-manager"

const clientWith = (send: Mock): SecretsManagerClient =>
    ({send}) as unknown as SecretsManagerClient

test('aws resolver throws error if key@[version/latest] format is not followed', async () => {
    const send = vi.fn()

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    //no @version suffix
    await expect(resolver.resolve("db")).rejects.toThrow()
    expect(send).not.toHaveBeenCalled()
})

test('aws resolver key@stage format fetches secret at that stage', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "s3cr3t"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("db@AWSPREVIOUS")).eq("s3cr3t")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "db", VersionStage: "AWSPREVIOUS"}}))
})

test('aws resolver key@latest omits VersionStage (AWSCURRENT default)', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "s3cr3t"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("db@latest")).eq("s3cr3t")
    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "db"}}))
})

test('aws resolver builds request from secret id', async () => {
    const send = vi.fn().mockResolvedValue({SecretString: "v"})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await resolver.resolve("prod/db/password@latest")

    expect(send).toHaveBeenCalledWith(expect.objectContaining({input: {SecretId: "prod/db/password"}}))
})

test('aws resolver decodes SecretBinary', async () => {
    const send = vi.fn().mockResolvedValue({SecretBinary: Buffer.from("binary-secret", "utf8")})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    expect(await resolver.resolve("db@latest")).eq("binary-secret")
})

test('aws resolver throws when secret doesn\'t exist', async () => {
    const send = vi.fn().mockRejectedValue(new Error("not found"))

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("db@latest")).rejects.toThrow()
})

test('aws resolver throws when secret has no value', async () => {
    const send = vi.fn().mockResolvedValue({})

    const resolver = AWSSecretsResolver({client: clientWith(send)})

    await expect(resolver.resolve("db@latest")).rejects.toThrow()
})
