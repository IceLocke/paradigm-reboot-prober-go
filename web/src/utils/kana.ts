export const KanaUtils = (() => {
  function toHiraganaCase(input: string) {
    return Array.from(input).map(char => {
      const code = char.charCodeAt(0)
      return String.fromCharCode(
        (0x30A1 <= code && code <= 0x30F6) ? code - 0x60 : code
      )
    }).join('')
  }

  function toKatakanaCase(input: string) {
    return Array.from(input).map(char => {
      const code = char.charCodeAt(0)
      return String.fromCharCode(
        (0x3041 <= code && code <= 0x3096) ? code + 0x60 : code
      )
    }).join('')
  }

  const ZENKANA_MAPPING: { [key: number]: number } = {
    65382: 12530,
    65383: 12449,
    65384: 12451,
    65385: 12453,
    65386: 12455,
    65387: 12457,
    65388: 12515,
    65389: 12517,
    65390: 12519,
    65391: 12483,
    65392: 12540,
    65393: 12450,
    65394: 12452,
    65395: 12454,
    65396: 12456,
    65397: 12458,
    65398: 12459,
    65399: 12461,
    65400: 12463,
    65401: 12465,
    65402: 12467,
    65403: 12469,
    65404: 12471,
    65405: 12473,
    65406: 12475,
    65407: 12477,
    65408: 12479,
    65409: 12481,
    65410: 12484,
    65411: 12486,
    65412: 12488,
    65413: 12490,
    65414: 12491,
    65415: 12492,
    65416: 12493,
    65417: 12494,
    65418: 12495,
    65419: 12498,
    65420: 12501,
    65421: 12504,
    65422: 12507,
    65423: 12510,
    65424: 12511,
    65425: 12512,
    65426: 12513,
    65427: 12514,
    65428: 12516,
    65429: 12518,
    65430: 12520,
    65431: 12521,
    65432: 12522,
    65433: 12523,
    65434: 12524,
    65435: 12525,
    65436: 12527,
    65437: 12531,
    65438: 12443,
    65439: 12444,
  }

  const HANKANA_MAPPING = Object.fromEntries(
    Object.entries(ZENKANA_MAPPING).map(([k, v]) => [v, +k])
  )

  function toHankanaCase(input: string) {
    const result = []
    for (let i = 0; i < input.length; i++) {
      const code = input.charCodeAt(i)
      if (HANKANA_MAPPING[code]) {
        result.push(HANKANA_MAPPING[code])
      } else if (0x30AB <= code && code <= 0x30C9) {
        result.push(HANKANA_MAPPING[code - 1], 0xFF9E)
      } else if (0x30CF <= code && code <= 0x30DD) {
        result.push(HANKANA_MAPPING[code - (code % 3)], [0xFF9E, 0xFF9F][(code % 3) - 1])
      } else {
        result.push(code)
      }
    }
    return String.fromCharCode(...result)
  }

  function toZenkanaCase(input: string) {
    const result = Array.from(input).map(char => {
      const code = char.charCodeAt(0)
      return String.fromCharCode(ZENKANA_MAPPING[code] || code)
    })
    return toPaddingCase(result.join(''))
  }

  function toPaddingCase(input: string) {
    const result = []
    for (let i = 0; i < input.length; i++) {
      const code = input.charCodeAt(i)
      const nextCode = input.charCodeAt(i + 1)
      let pushed = false

      if (
        (0x304B <= code && code <= 0x3062 && code % 2 === 1) ||
        (0x30AB <= code && code <= 0x30C2 && code % 2 === 1) ||
        (0x3064 <= code && code <= 0x3069 && code % 2 === 0) ||
        (0x30C4 <= code && code <= 0x30C9 && code % 2 === 0)
      ) {
        const added = { 0x309B: 1 } [nextCode] || 0
        result.push(code + added)
        if (added) i++
        pushed = true
      }

      if (
        !pushed &&
        (
          (0x306F <= code && code <= 0x307D && code % 3 === 0) ||
          (0x30CF <= code && code <= 0x30DD && code % 3 === 0)
        )
      ) {
        const added = { 0x309B: 1, 0x309C: 2 } [nextCode] || 0
        result.push(code + added)
        if (added) i++
        pushed = true
      }

      if (!pushed) {
        result.push(code)
      }
    }
    return String.fromCharCode(...result)
  }

  return {
    toHiraganaCase,
    toKatakanaCase,
    toHankanaCase,
    toZenkanaCase,
    toPaddingCase,
  }
})()

export default KanaUtils