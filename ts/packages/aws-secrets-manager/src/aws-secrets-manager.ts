import {SecretsManagerClient, GetSecretValueCommand} from "@aws-sdk/client-secrets-manager"
import type {Resolver} from "@comfig/core"

export interface AWSSecretsResolverOptions {
    region?: string
    client?: SecretsManagerClient
}

export const AWSSecretsResolver = (options: AWSSecretsResolverOptions = {}, prefixOverride?: string): Resolver => {
    const client = options.client ?? new SecretsManagerClient(options.region ? {region: options.region} : {})

    return {
        prefix: prefixOverride ?? "aws",
        resolve: async (value): Promise<string> => {
            const {secretId, selector} = parseReference(value)

            const response = await client.send(new GetSecretValueCommand({
                SecretId: secretId,
                ...selector
            }))

            if (response.SecretString !== undefined) {
                return response.SecretString
            }

            if (response.SecretBinary !== undefined) {
                return Buffer.from(response.SecretBinary).toString("utf8")
            }

            throw new Error(`Secret ${value} has no value`)
        }
    } satisfies Resolver
}

interface VersionSelector {
    VersionId?: string
    VersionStage?: string
}

interface SecretReference {
    secretId: string
    selector?: VersionSelector
}

const parseReference = (value: string): SecretReference => {
    if (value === "") {
        throw new Error("AWS secret can't be empty")
    }

    return parseVersionIdReference(value) ?? parseVersionStageReference(value) ?? parseWithNoSelector(value)
}

const parseVersionIdReference = (value: string): SecretReference | undefined => {
    const [rawSecretId, rawVersionId, ...extraVersionIds] = value.split("#")
    if (rawVersionId === undefined) return undefined

    if (extraVersionIds.length > 0) {
        throw new Error("AWS secret reference has multiple version ID selectors")
    }

    if (rawSecretId === "") {
        throw new Error("AWS secret reference has an empty secret ID")
    }

    if (rawSecretId.includes("@")) {
        throw new Error("AWS secret IDs containing @ must use %40 when selecting a version ID")
    }

    if (rawVersionId === "") {
        throw new Error("AWS secret reference has an empty version ID")
    }

    return {
        secretId: decode(rawSecretId, "secret ID"),
        selector: {VersionId: decode(rawVersionId, "version ID")}
    }
}

const parseVersionStageReference = (value: string): SecretReference | undefined => {
    const [rawStageSecretId, rawVersionStage, ...extraVersionStages] = value.split("@")
    if (rawVersionStage === undefined) return undefined

    if (rawStageSecretId === "") {
        throw new Error("AWS secret reference has an empty secret ID")
    }

    if (extraVersionStages.length > 0) {
        throw new Error("AWS secret IDs containing @ must use %40 when selecting a version stage")
    }

    if (rawVersionStage === "") {
        throw new Error("AWS secret reference has an empty version stage")
    }

    return {
        secretId: decode(rawStageSecretId, "secret ID"),
        selector: {VersionStage: decode(rawVersionStage, "version stage")}
    }
}

const parseWithNoSelector = (value: string): SecretReference => ({
    secretId: decode(value, "secret ID")
})

const decode = (value: string, description: string): string => {
    try {
        return decodeURIComponent(value)
    } catch {
        throw new Error(`AWS secret reference has invalid percent-encoded ${description}`)
    }
}
