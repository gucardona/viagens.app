// Flag emoji for an ISO 3166-1 alpha-2 code ("pt" -> 🇵🇹).
export const flag = (code: string) =>
  String.fromCodePoint(...[...code.toUpperCase()].map((c) => 0x1f1e6 + c.charCodeAt(0) - 65))
