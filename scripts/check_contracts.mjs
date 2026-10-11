import SwaggerParser from "@apidevtools/swagger-parser";
import { Parser } from "@asyncapi/parser";
import { readFile } from "node:fs/promises";
import path from "node:path";

await SwaggerParser.validate("contracts/openapi.json");
const parser = new Parser();
const { document, diagnostics } = await parser.parse(await readFile("contracts/asyncapi.json", "utf8"), { source: path.resolve("contracts/asyncapi.json") });
const errors = diagnostics.filter(d => d.severity === 0);
if (!document || errors.length) throw new Error(JSON.stringify(errors));
console.log("OpenAPI 3.1 and AsyncAPI 3.0 validation passed.");
