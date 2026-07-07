import {z} from "zod";

export const expandable = <T extends z.ZodTypeAny>(target: T, parse: (data: string) => z.input<T> = (data: string) => JSON.parse(data)) => (
    z.codec(z.string(), target, {
        decode: (s, ctx) => {
            try { return parse(s) as z.input<T> }
            catch (e) {ctx.issues.push({
                code: "invalid_format", format: "custom", input: s, message: String(e) });
                return z.NEVER
            }
        },
        encode: (v) => {
            throw Error("encode is not supported")
        }
    })
)
