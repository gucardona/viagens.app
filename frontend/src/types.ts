export type Kind = 'flight' | 'lodging' | 'car' | 'transport' | 'restaurant' | 'activity' | 'event' | 'note'

export type Trip = {
  id: number
  name: string
  destination: string
  start_date: string
  end_date: string
  notes: string
  created_at: string
  updated_at: string
}

export type Item = {
  id: number
  trip_id: number
  kind: Kind
  title: string
  start_date: string
  start_time: string
  end_date: string
  end_time: string
  address: string
  lat: number | null
  lng: number | null
  confirmation_code: string
  notes: string
  created_at: string
  updated_at: string
}

export type Attachment = {
  id: number
  trip_id: number
  item_id: number | null
  filename: string
  content_type: string
  size: number
  created_at: string
}

export type TripDetail = { trip: Trip; items: Item[]; attachments: Attachment[] }
