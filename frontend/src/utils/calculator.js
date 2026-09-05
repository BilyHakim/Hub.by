// Evaluate arithmetic without executing user input as JavaScript.
export function calculate(expression) {
  const source = expression.replace(/\s/g, '').replace(/×/g, '*').replace(/÷/g, '/').replace(/−/g, '-').replace(/,/g, '.')
  const tokens = source.match(/\d+(?:\.\d*)?|\.\d+|[+*/-]/g) || []
  if (!source || tokens.join('') !== source) throw new Error('Perhitungan belum lengkap.')
  let index = 0
  function number() {
    let sign = 1
    if (tokens[index] === '-' || tokens[index] === '+') sign = tokens[index++] === '-' ? -1 : 1
    const token = tokens[index++]
    if (!token || !/^(?:\d|\.)/.test(token)) throw new Error('Perhitungan belum lengkap.')
    return sign * Number(token)
  }
  function product() {
    let value = number()
    while (tokens[index] === '*' || tokens[index] === '/') {
      const operator = tokens[index++]
      const right = number()
      if (operator === '/' && right === 0) throw new Error('Tidak bisa membagi dengan nol.')
      value = operator === '*' ? value * right : value / right
    }
    return value
  }
  let value = product()
  while (index < tokens.length) {
    const operator = tokens[index++]
    if (operator !== '+' && operator !== '-') throw new Error('Perhitungan belum lengkap.')
    const right = product()
    value = operator === '+' ? value + right : value - right
  }
  if (!Number.isFinite(value) || Math.abs(value) > Number.MAX_SAFE_INTEGER) throw new Error('Hasil terlalu besar.')
  return value
}
