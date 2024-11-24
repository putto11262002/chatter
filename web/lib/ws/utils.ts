export function base64Decode(str: string): string {
  return atob(str);
}

export function base64Encode(str: string): string {
  return btoa(str);
}

export function generateCorrelationID(): number {
  return Math.floor(Math.random() * 65536);
}
