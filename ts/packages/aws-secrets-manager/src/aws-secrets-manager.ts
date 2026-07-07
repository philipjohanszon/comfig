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
            const [secret, version] = value.split("@")

            if (!version) {
                throw Error(`Secret "${value}" must use the <name>@<version> format, e.g. "db@latest" or "db@3"`)
            }

            const response = await client.send(new GetSecretValueCommand({
                SecretId: secret,
                ...(version == "latest" ? {} : {VersionStage: version})
            }))

            if (response.SecretString !== undefined) {
                return response.SecretString
            }

            if (response.SecretBinary !== undefined) {
                return Buffer.from(response.SecretBinary).toString("utf8")
            }

            throw Error(`Secret ${secret}@${version} has no value`)
        }
    } satisfies Resolver
}
