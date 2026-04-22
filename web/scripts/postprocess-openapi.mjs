#!/usr/bin/env node
/**
 * Post-processes the OpenAPI 3 document produced by `swagger2openapi`.
 *
 * `swaggo/swag` emits OpenAPI 2.0 with the Swagger-2 extension `x-nullable`
 * (as a *string* "true", because swag treats all `extensions:"k=v"` tag values
 * as strings). `swagger2openapi` preserves `x-` extensions verbatim but does
 * not translate them to OpenAPI 3.0's native `nullable: true` boolean.
 *
 * This script walks the document and, wherever it sees `x-nullable` equal to
 * the string "true" or the boolean true, promotes it to `nullable: true` and
 * removes the extension — producing a spec that `openapi-typescript` can
 * understand natively (generating `T | null` for nullable fields).
 */
import fs from 'node:fs'
import path from 'node:path'

const target = path.resolve(process.cwd(), 'src/api/openapi3.json')
const raw = fs.readFileSync(target, 'utf8')
const doc = JSON.parse(raw)

let touched = 0

function walk(node) {
  if (!node || typeof node !== 'object') return
  if (Array.isArray(node)) {
    for (const item of node) walk(item)
    return
  }
  if ('x-nullable' in node) {
    const v = node['x-nullable']
    if (v === true || v === 'true') {
      node.nullable = true
      touched++
    }
    delete node['x-nullable']
  }
  for (const key of Object.keys(node)) walk(node[key])
}

walk(doc)
fs.writeFileSync(target, JSON.stringify(doc, null, 2) + '\n')
console.log(`✔ normalized ${touched} x-nullable entr${touched === 1 ? 'y' : 'ies'} → nullable:true in ${path.relative(process.cwd(), target)}`)
