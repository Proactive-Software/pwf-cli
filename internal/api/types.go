package api

// ItemContact is a contact summary returned in list responses.
type ItemContact struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
}

// Item is the list-response shape from /projectitems.
type Item struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Code         string        `json:"code"`
	UniqueToken  string        `json:"uniquetoken"`
	ProjectID    int           `json:"projectid"`
	ProjectTitle string        `json:"projecttitle"`
	PhaseName    string        `json:"phasename"`
	WorkstageID  int           `json:"activeworkstageid"`
	Contacts     []ItemContact `json:"contacts"`
}

// ItemDetail is the full single-item response from /projectitems/{id}.
type ItemDetail struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Code              string `json:"code"`
	UniqueToken       string `json:"uniquetoken"`
	ProjectID         int    `json:"projectid"`
	ProjectTitle      string `json:"projecttitle"`
	PhaseName         string `json:"phasename"`
	ItemCollectionID  int    `json:"itemcollectionid"`
	WorkstageID       int    `json:"activeworkstageid"`
	Description       string `json:"description"`
	Contacts          []struct {
		ID        int    `json:"id"`
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
	} `json:"contacts"`
}

// listEnvelope wraps paginated list responses.
type listEnvelope[T any] struct {
	Status string `json:"status"`
	Data   []T    `json:"data"`
}

// singleEnvelope wraps single-object responses.
type singleEnvelope[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
}

// Phase is returned from GET /projects/{id}/phases.
type Phase struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Workstage is returned from GET /settings/workstages/item.
type Workstage struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	WorkState string `json:"workstate"`
	Color     string `json:"color"`
}

// errorEnvelope is returned on errors.
type errorEnvelope struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Data    []string `json:"data"`
}
