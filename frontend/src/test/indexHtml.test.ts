import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('index.html', () => {
  const html = readFileSync(resolve(__dirname, '../../index.html'), 'utf8')

  it('declares Spanish as the document language', () => {
    expect(html).toMatch(/<html lang="es">/)
  })

  it('keeps the product name as the page title', () => {
    expect(html).toContain('<title>Energy Console</title>')
  })
})
