import { describe, expect, it } from 'vitest'
import { parseCSV, parseTakeout } from './takeout'

describe('parseTakeout', () => {
  it('reads starred places from GeoJSON', () => {
    const geo = JSON.stringify({
      type: 'FeatureCollection',
      features: [
        { geometry: { coordinates: [-9.2032, 38.6975] }, properties: { google_maps_url: 'u1', location: { name: 'Pastéis de Belém' }, Comment: 'nata' } },
        { geometry: { coordinates: [0, 0] }, properties: { Title: 'Sem local', google_maps_url: 'u2' } },
      ],
    })
    expect(parseTakeout('Saved Places.json', geo)).toEqual([
      { title: 'Pastéis de Belém', notes: 'nata', url: 'u1', lat: 38.6975, lng: -9.2032 },
      { title: 'Sem local', notes: '', url: 'u2', lat: null, lng: null },
    ])
  })

  it('reads a saved list CSV', () => {
    const csv = 'Title,Note,URL,Tags,Comment\n"Tasca, do Chico","ir ""cedo""",https://maps/x,,\n\nCastelo,,https://maps/y,,\n'
    expect(parseTakeout('Quero ir.csv', csv)).toEqual([
      { title: 'Tasca, do Chico', notes: 'ir "cedo"', url: 'https://maps/x', lat: null, lng: null },
      { title: 'Castelo', notes: '', url: 'https://maps/y', lat: null, lng: null },
    ])
  })
})

describe('parseCSV', () => {
  it('handles quotes, commas, CRLF and newlines inside quotes', () => {
    expect(parseCSV('a,"b,c"\r\n"d\ne",f')).toEqual([['a', 'b,c'], ['d\ne', 'f']])
  })
})
