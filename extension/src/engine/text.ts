export function normalise(value: string): string {
  return value
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, ' ')
    .trim()
}

export function hasPhrase(normalisedText: string, phrase: string): boolean {
  const p = normalise(phrase)
  if (!p) return false
  if (` ${normalisedText} `.includes(` ${p} `)) return true
  if (!p.includes(' ')) return false
  const joined = p.replace(/ /g, '')
  return normalisedText.split(' ').some((word) => word === joined || word.startsWith(joined))
}
