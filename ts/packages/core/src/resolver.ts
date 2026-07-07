export type ResolverFactory<T> = (raw: T) => Promise<Resolver> | Resolver

export interface Resolver {
    prefix: string,
    resolve: (value: string) => Promise<string> | string
}

export const EnvResolver = (prefixOverride?: string): Resolver => ({
    prefix: prefixOverride || "env",
    resolve: (value): string => {
        const variable = process.env[value]

        if (variable === undefined) {
            throw Error(`Could not get environment variable ${value}`)
        }

        return variable
    }
} satisfies Resolver)