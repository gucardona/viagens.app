import type { Item } from './types'

export function mapLinks(i: Pick<Item, 'lat' | 'lng' | 'address'>) {
  if (i.lat != null && i.lng != null) {
    return {
      google: `https://www.google.com/maps/search/?api=1&query=${i.lat},${i.lng}`,
      waze: `https://waze.com/ul?ll=${i.lat},${i.lng}&navigate=yes`,
    }
  }
  if (i.address) {
    const q = encodeURIComponent(i.address)
    return { google: `https://www.google.com/maps/search/?api=1&query=${q}`, waze: `https://waze.com/ul?q=${q}&navigate=yes` }
  }
  return null
}
