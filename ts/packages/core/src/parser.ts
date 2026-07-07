export interface Parser {
    extension: string,
    parse: (raw: string) => Record<string, unknown>
}

export const JSONParser = {
    extension: "json",
    parse: (raw: string) => JSON.parse(raw)
} satisfies Parser