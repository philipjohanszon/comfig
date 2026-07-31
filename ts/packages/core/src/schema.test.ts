import {expect, test} from "vitest";
import {z} from "zod";
import {expandable} from "./schema";

const connection = expandable(z.object({
    host: z.string(),
    port: z.number(),
}))

test("expandable accepts an already parsed value", () => {
    expect(connection.parse({ host: "localhost", port: 5432 })).toEqual({
        host: "localhost",
        port: 5432,
    })
})

test("expandable rejects a string that cannot be decoded", () => {
    expect(connection.safeParse("{not valid JSON}").success).toBe(false)
})

test("expandable decodes JSON strings with its default parser", () => {
    expect(connection.parse('{"host":"localhost","port":5432}')).toEqual({
        host: "localhost",
        port: 5432,
    })
})
