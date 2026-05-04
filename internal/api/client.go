package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func unescapeItems(items []Item) {
	for i := range items {
		items[i].Name = html.UnescapeString(items[i].Name)
	}
}

func unescapeDetail(d *ItemDetail) {
	d.Name = html.UnescapeString(d.Name)
	d.ProjectTitle = html.UnescapeString(d.ProjectTitle)
}

// Client wraps the ProWorkflow v4 API.
type Client struct {
	base   string
	apiKey string
	http   *http.Client
}

func New(base, apiKey string) *Client {
	return &Client{
		base:   strings.TrimRight(base, "/"),
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

func (c *Client) get(path string, params url.Values, out any) error {
	u := c.base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return c.do(req, out)
}

func (c *Client) put(path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, nil)
}

func (c *Client) post(path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		var e errorEnvelope
		_ = json.Unmarshal(raw, &e)
		if e.Message != "" {
			return fmt.Errorf("API %d: %s", resp.StatusCode, e.Message)
		}
		if len(e.Data) > 0 {
			return fmt.Errorf("API %d: %s", resp.StatusCode, strings.Join(e.Data, "; "))
		}
		return fmt.Errorf("API %d", resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// MyItems returns active items assigned to the authenticated user.
func (c *Client) MyItems() ([]Item, error) {
	params := url.Values{
		"contacts":   {"me"},
		"status":     {"active"},
		"fields":     {"name,code,phasename,project,workstageid,priority,dates"},
		"pagesize":   {"100"},
		"pagenumber": {"1"},
	}
	var env listEnvelope[Item]
	if err := c.get("/projectitems", params, &env); err != nil {
		return nil, err
	}
	unescapeItems(env.Data)
	return env.Data, nil
}

// ActiveItems returns all active items (for --all browsing).
func (c *Client) ActiveItems() ([]Item, error) {
	params := url.Values{
		"status":     {"active"},
		"fields":     {"name,code,phasename,project,workstageid,priority,dates"},
		"pagesize":   {"100"},
		"pagenumber": {"1"},
	}
	var env listEnvelope[Item]
	if err := c.get("/projectitems", params, &env); err != nil {
		return nil, err
	}
	unescapeItems(env.Data)
	return env.Data, nil
}

// ProjectItems returns active items in a specific project.
func (c *Client) ProjectItems(projectID int) ([]Item, error) {
	params := url.Values{
		"projectid":  {fmt.Sprintf("%d", projectID)},
		"status":     {"active"},
		"fields":     {"name,code,phasename,project,workstageid,contacts"},
		"pagesize":   {"200"},
		"pagenumber": {"1"},
	}
	var env listEnvelope[Item]
	if err := c.get("/projectitems", params, &env); err != nil {
		return nil, err
	}
	unescapeItems(env.Data)
	return env.Data, nil
}

// InboxItems returns active unassigned items in a specific project.
func (c *Client) InboxItems(projectID int) ([]Item, error) {
	params := url.Values{
		"projectid":  {fmt.Sprintf("%d", projectID)},
		"contacts":   {"unassigned"},
		"status":     {"active"},
		"fields":     {"name,code,phasename,project,workstageid"},
		"pagesize":   {"200"},
		"pagenumber": {"1"},
	}
	var env listEnvelope[Item]
	if err := c.get("/projectitems", params, &env); err != nil {
		return nil, err
	}
	unescapeItems(env.Data)
	return env.Data, nil
}

// SearchItems searches items by name across the account.
func (c *Client) SearchItems(query string) ([]Item, error) {
	params := url.Values{
		"search":     {query},
		"status":     {"active"},
		"fields":     {"name,code,phasename,project,workstageid,contacts"},
		"pagesize":   {"50"},
		"pagenumber": {"1"},
	}
	var env listEnvelope[Item]
	if err := c.get("/projectitems", params, &env); err != nil {
		return nil, err
	}
	unescapeItems(env.Data)
	return env.Data, nil
}

// AssignContact assigns a contact to an item (replaces existing contacts).
func (c *Client) AssignContact(itemID, contactID int) error {
	return c.put(fmt.Sprintf("/projectitems/%d", itemID), map[string]any{
		"contactid": []int{contactID},
	})
}

// GetItem fetches a single item by ID (includes uniquetoken).
func (c *Client) GetItem(id int) (*ItemDetail, error) {
	var env singleEnvelope[ItemDetail]
	if err := c.get(fmt.Sprintf("/projectitems/%d", id), nil, &env); err != nil {
		return nil, err
	}
	unescapeDetail(&env.Data)
	return &env.Data, nil
}

// SetWorkstage updates the active workstage for an item.
func (c *Client) SetWorkstage(itemID, workstageID int) error {
	return c.put(fmt.Sprintf("/projectitems/%d/workstage", itemID), map[string]int{
		"activeworkstageid": workstageID,
	})
}

// CreateItem creates a new item in the given project phase.
func (c *Client) CreateItem(projectID, itemCollectionID, contactID int, name string) (*ItemDetail, error) {
	payload := map[string]any{
		"name":             name,
		"itemcollectionid": itemCollectionID,
		"contactid":        []int{contactID},
		"activeworkstate":  "active",
	}
	var env singleEnvelope[ItemDetail]
	if err := c.post(fmt.Sprintf("/projects/%d/items", projectID), payload, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetPhases fetches all phases for a project.
func (c *Client) GetPhases(projectID int) ([]Phase, error) {
	params := url.Values{"fields": {"id,name"}}
	var env listEnvelope[Phase]
	if err := c.get(fmt.Sprintf("/projects/%d/phases", projectID), params, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// GetItemWorkstages fetches all item workstages from /settings/workstages/item.
func (c *Client) GetItemWorkstages() ([]Workstage, error) {
	var env listEnvelope[Workstage]
	if err := c.get("/settings/workstages/item", nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// Me returns the authenticated user's contact record.
func (c *Client) Me() (map[string]any, error) {
	var env singleEnvelope[map[string]any]
	if err := c.get("/me", nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}
