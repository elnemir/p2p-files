import SwaggerParser from "@apidevtools/swagger-parser";
import { Parser } from "@asyncapi/parser";
import { readFile, readdir } from "node:fs/promises";
import Ajv from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import path from "node:path";

await SwaggerParser.validate("contracts/openapi.json");
const schemas = new Ajv({ strict: true });
addFormats(schemas);
for (const name of await readdir("contracts/schemas")) {
  if (!name.endsWith(".json")) continue;
  const schema = JSON.parse(await readFile("contracts/schemas/" + name, "utf8"));
  schemas.addSchema(schema);
}
for (const name of await readdir("contracts/schemas")) {
  if (!name.endsWith(".json")) continue;
  const schema = JSON.parse(await readFile("contracts/schemas/" + name, "utf8"));
  if (!schemas.getSchema(schema.$id)) throw new Error("Invalid standalone schema: " + name);
}
const parser = new Parser();
const { document, diagnostics } = await parser.parse(await readFile("contracts/asyncapi.json", "utf8"), { source: path.resolve("contracts/asyncapi.json") });
const errors = diagnostics.filter(d => d.severity === 0);
if (!document || errors.length) throw new Error(JSON.stringify(errors));
console.log("OpenAPI 3.1 and AsyncAPI 3.0 validation passed.");
