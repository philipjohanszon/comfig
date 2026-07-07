import { test, expect } from 'vitest'
import {JSONParser} from "./parser";

test('JSON parser can read json', () => {
    const valid = `
    {
        "project": "test-project",
        "instances": 2,
        "secret": "sm://this-is-secret:1",
        "flags": {
            "debugEndpoints": true,
            "noAuth": false
        }
    } 
    `

    expect(JSONParser.parse(valid)).toEqual({
        project: "test-project",
        instances: 2,
        secret: "sm://this-is-secret:1",
        flags: {
            debugEndpoints: true,
            noAuth: false
        }
    })
})

test('JSON parser throws on invalid json', () => {
    const invalid = 'th"is2 is not valid'

    expect(() => JSONParser.parse(invalid)).to.throw()
})