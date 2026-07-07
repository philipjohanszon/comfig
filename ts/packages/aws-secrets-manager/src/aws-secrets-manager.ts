import {SecretsManagerClient, GetSecretValueCommand} from "@aws-sdk/client-secrets-manager"
import type {Resolver} from "@comfig/core"

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

export interface AWSSecretsResolverOptions {
    region?: string
    client?: SecretsManagerClient
}

export const AWSSecretsResolver = (options: AWSSecretsResolverOptions = {}, prefixOverride?: string): Resolver => {
    const client = options.client ?? new SecretsManagerClient(options.region ? {region: options.region} : {})

    return {
        prefix: prefixOverride ?? "aws",
        resolve: async (value): Promise<string> => {
            const [secretId, version] = value.split("@")

            const isPinned = version !== undefined && version !== "" && version !== "latest"

            const isUUID = isPinned ? UUID.test(version) : false

            const response = await client.send(new GetSecretValueCommand({
                SecretId: secretId,
                ...(isPinned && isUUID ? {VersionId: version} : {}),
                ...(isPinned && !isUUID ? {VersionStage: version} : {})
            }))

            if (response.SecretString !== undefined) {
                return response.SecretString
            }

            if (response.SecretBinary !== undefined) {
                return Buffer.from(response.SecretBinary).toString("utf8")
            }

            throw Error(`Secret ${value} has no value`)
        }
    } satisfies Resolver
}
