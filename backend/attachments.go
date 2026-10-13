package main

type Attachment struct {
	ID          int64  `json:"id"`
	TripID      int64  `json:"trip_id"`
	ItemID      *int64 `json:"item_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	CreatedAt   string `json:"created_at"`
}

const attCols = `id, trip_id, item_id, filename, content_type, size, created_at`

func (a *App) listAttachments(tripID int64) ([]Attachment, error) {
	rows, err := a.db.Query(`SELECT `+attCols+` FROM attachments WHERE trip_id = ? ORDER BY id`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	atts := []Attachment{}
	for rows.Next() {
		var at Attachment
		if err := rows.Scan(&at.ID, &at.TripID, &at.ItemID, &at.Filename, &at.ContentType, &at.Size, &at.CreatedAt); err != nil {
			return nil, err
		}
		atts = append(atts, at)
	}
	return atts, rows.Err()
}
