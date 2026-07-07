import type {ResolverFactory, Resolver} from "./resolver.js";
import {JSONParser, type Parser} from "./parser.js";
import {z} from "zod";
import {FileSystemSource, type Source} from "./source.js";
import {environment} from "./environment.js";

export class Comfig<T extends z.ZodObject> {
    private source?: Source
    private parser: Parser = JSONParser
    private resolverFactories: ResolverFactory<z.input<T>>[] = []

    constructor(private schema: T) {}

    public useResolver(factory: ResolverFactory<z.input<T>>): Comfig<T> {
        this.resolverFactories.push(factory)
        return this
    }

    public usePath(path: string): Comfig<T> {
        this.source = FileSystemSource(path)
        return this
    }

    public useSource(source: Source): Comfig<T> {
        this.source = source
        return this
    }

    public useParser(parser: Parser): Comfig<T> {
        this.parser = parser
        return this
    }

    public async load(): Promise<z.output<T>> {
        if (!this.source) {
            throw Error("No path was provided for the configuration. Please add a path with usePath() or add another source with useSource()")
        }

        const fileContents = await this.source.getConfiguration(environment, this.parser.extension)
        const raw = this.parser.parse(fileContents)
        const resolvers = await this.createResolvers(raw as z.input<T>)
        const resolved = await resolveTree(raw, resolvers)

        return this.schema.parse(resolved)
    }

    private async createResolvers(configuration: z.input<T>): Promise<Record<string, Resolver>> {
        const createdResolvers = await Promise.all(
            this.resolverFactories.map(
                (resolverFactory) => resolverFactory(configuration)
            )
        )

        const resolvers: Record<string, Resolver> = {}
        const duplicates: string[] = []
        for (const resolver of  createdResolvers) {
            if ( resolvers[resolver.prefix]) {
                duplicates.push(resolver.prefix)
            } else {
                resolvers[resolver.prefix] = resolver
            }
        }

        if (duplicates.length > 0) {
            throw Error(`Duplicate resolver prefix${duplicates.length > 1 ? "es" : ""} found: ${duplicates.map((prefix) => `${prefix}://*`).join(", ")}`)
        }

        return resolvers
    }
}

const resolveTree = async (
    node: unknown,
    resolvers: Record<string, Resolver>,
    path = "$",
): Promise<unknown> => {
    if (typeof node === "string") {
        const { prefix, rest } = split(node)
        if (!prefix) return node
        const resolver = resolvers[prefix]
        if (!resolver) return node
        try {
            return await resolver.resolve(rest)
        } catch (cause) {
            throw new Error(`failed to resolve "${node}" at ${path}`, { cause })
        }
    }

    if (Array.isArray(node)) {
        return Promise.all(node.map((el, i) => resolveTree(el, resolvers, `${path}[${i}]`)))
    }

    if (node !== null && typeof node === "object") {
        const resolved = await Promise.all(
            Object.entries(node as Record<string, unknown>).map(
                async ([k, v]) => [k, await resolveTree(v, resolvers, `${path}.${k}`)] as const,
            ),
        )
        return Object.fromEntries(resolved)
    }

    return node
}

const split = (value: string): { prefix?: string; rest: string } => {
    if (!value.includes("://")) return { rest: value }
    const [prefix, ...rest] = value.split("://")
    return { prefix, rest: rest.join("://") }
}