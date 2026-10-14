package main

import (
	"fmt"
	"testing"
)

func TestChecklist(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	url := fmt.Sprintf("%s/api/trips/%d/checklist", srv.URL, trip.ID)

	a := decode[CheckItem](t, do(t, c, "POST", url, map[string]any{"text": "Passaporte"}))
	do(t, c, "POST", url, map[string]any{"text": "Adaptador de tomada"})
	if res := do(t, c, "POST", url, map[string]any{"text": "  "}); res.StatusCode != 400 {
		t.Fatalf("empty: %d", res.StatusCode)
	}
	res := do(t, c, "PUT", fmt.Sprintf("%s/api/checklist/%d", srv.URL, a.ID), map[string]any{"text": "Passaporte", "done": true})
	if got := decode[CheckItem](t, res); res.StatusCode != 200 || !got.Done {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	list := decode[[]CheckItem](t, do(t, c, "GET", url, nil))
	if len(list) != 2 || list[0].Text != "Passaporte" || !list[0].Done || list[1].Done {
		t.Fatalf("%+v", list)
	}

	// Copy into another trip: texts come unchecked and existing ones are not repeated.
	other := createTrip(t, c, srv)
	otherURL := fmt.Sprintf("%s/api/trips/%d/checklist", srv.URL, other.ID)
	do(t, c, "POST", otherURL, map[string]any{"text": "passaporte"})
	res = do(t, c, "POST", otherURL+"/copy", map[string]any{"from_trip_id": trip.ID})
	copied := decode[[]CheckItem](t, res)
	if res.StatusCode != 200 || len(copied) != 2 || copied[1].Text != "Adaptador de tomada" || copied[1].Done {
		t.Fatalf("%d %+v", res.StatusCode, copied)
	}

	do(t, c, "DELETE", fmt.Sprintf("%s/api/checklist/%d", srv.URL, a.ID), nil)
	if list := decode[[]CheckItem](t, do(t, c, "GET", url, nil)); len(list) != 1 {
		t.Fatalf("after delete %+v", list)
	}
}
