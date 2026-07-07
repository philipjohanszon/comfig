import {SecretManagerServiceClient} from "@google-cloud/secret-manager"
import type {Resolver} from "@comfig/core"

export interface GCPSecretResolverOptions {
    projectId: string
    client?: SecretManagerServiceClient
}

export const GCPSecretResolver = (options: GCPSecretResolverOptions, prefixOverride?: string): Resolver => {
    if (!options.projectId) {
        throw Error('Cannot resolve secrets without a projectId. Provide options.projectId')
    }

    const client = options.client ?? new SecretManagerServiceClient()

    return {
        prefix: prefixOverride ?? "gcp",
        resolve: async (value): Promise<string> => {
            const name = resolveName(value, options.projectId)

            const [response] = await client.accessSecretVersion({name})
            const data = response?.payload?.data

            if (data === undefined || data === null) {
                throw Error(`Secret ${name} has no payload`)
            }

            return typeof data === "string" ? data : Buffer.from(data).toString("utf8")
        }
    } satisfies Resolver
}

const resolveName = (value: string, projectId: string): string => {
    const [secret, version] = value.split("@")

    if (!version) {
        throw Error(`Secret "${value}" must use the <name>@<version> format, e.g. "db@latest" or "db@3"`)
    }

    return `projects/${projectId}/secrets/${secret}/versions/${version}`
}
