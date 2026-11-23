package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Proactive-Software/pwf-cli/internal/api"
	"github.com/Proactive-Software/pwf-cli/internal/auth"
	"github.com/Proactive-Software/pwf-cli/internal/config"
	"github.com/Proactive-Software/pwf-cli/internal/keyring"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ServeStdio runs the MCP server using newline-delimited JSON-RPC on stdio.
func ServeStdio(input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)
	encoder := json.NewEncoder(output)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var req request
			if jsonErr := json.Unmarshal(line, &req); jsonErr == nil {
				resp, shouldReply := handleRequest(req)
				if shouldReply {
					if encErr := encoder.Encode(resp); encErr != nil {
						return encErr
					}
				}
			} else if strings.TrimSpace(string(line)) != "" {
				resp := response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "Parse error"}}
				if encErr := encoder.Encode(resp); encErr != nil {
					return encErr
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func handleRequest(req request) (response, bool) {
	resp := response{JSONRPC: "2.0"}
	if len(req.ID) > 0 {
		var id any
		if err := json.Unmarshal(req.ID, &id); err == nil {
			resp.ID = id
		}
	}
	hasID := len(req.ID) > 0

	switch req.Method {
	case "initialize":
		if hasID {
			protocolVersion := "2024-11-05"
			var params struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			if err := json.Unmarshal(req.Params, &params); err == nil && params.ProtocolVersion != "" {
				protocolVersion = params.ProtocolVersion
			}
			resp.Result = map[string]any{
				"protocolVersion": protocolVersion,
				"serverInfo": map[string]any{
					"name":    "pwf",
					"version": "1.0.0",
				},
				"capabilities": map[string]any{"tools": map[string]any{"listChanged": false}},
			}
		}
	case "ping":
		if hasID {
			resp.Result = map[string]any{}
		}
	case "notifications/initialized", "notifications/cancelled":
		return response{}, false
	case "tools/list":
		if hasID {
			resp.Result = map[string]any{"tools": tools()}
		}
	case "tools/call":
		if hasID {
			resp.Result = callTool(req.Params)
		}
	default:
		if hasID {
			resp.Error = &rpcError{Code: -32601, Message: "Method not found"}
		} else {
			return response{}, false
		}
	}
	return resp, hasID
}

func tools() []tool {
	return []tool{
		{Name: "list_items", Description: "List active ProWorkflow items.", InputSchema: objectSchema(map[string]any{
			"all": map[string]any{"type": "boolean", "description": "List all active account items instead of only items assigned to you."},
		})},
		{Name: "get_active_item", Description: "Get the currently active item.", InputSchema: objectSchema(nil)},
		{Name: "set_active_item", Description: "Set an item as active by ID.", InputSchema: objectSchema(map[string]any{
			"item_id": map[string]any{"type": "integer", "description": "ProWorkflow item ID."},
		}, "item_id")},
		{Name: "search_items", Description: "Search active items by name or code.", InputSchema: objectSchema(map[string]any{
			"query": map[string]any{"type": "string", "description": "Search text."},
		}, "query")},
		{Name: "get_item", Description: "Get an item's full details.", InputSchema: objectSchema(map[string]any{
			"item_id": map[string]any{"type": "integer", "description": "ProWorkflow item ID."},
		}, "item_id")},
		{Name: "set_workstage", Description: "Move an item to a workstage.", InputSchema: objectSchema(map[string]any{
			"workstage": map[string]any{"type": "string", "description": "Exact or fuzzy workstage name."},
			"item_id":   map[string]any{"type": "integer", "description": "Optional item ID. Defaults to the active item."},
		}, "workstage")},
		{Name: "create_item", Description: "Create an item in a project and phase.", InputSchema: objectSchema(map[string]any{
			"title":     map[string]any{"type": "string", "description": "Item title."},
			"project_id": map[string]any{"type": "integer", "description": "Optional project ID. Defaults to the active item project."},
			"phase_id":   map[string]any{"type": "integer", "description": "Optional phase/collection ID. Defaults to the active item phase."},
			"phase_name": map[string]any{"type": "string", "description": "Optional fuzzy phase name. Overrides phase_id."},
		}, "title")},
		{Name: "clear_active_item", Description: "Clear the currently active item.", InputSchema: objectSchema(nil)},
	}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func callTool(raw json.RawMessage) any {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return textToolError(fmt.Sprintf("invalid tools/call params: %v", err))
	}

	var result any
	var err error
	switch params.Name {
	case "list_items":
		result, err = listItems(params.Arguments["all"] == true)
	case "get_active_item":
		result, err = getActiveItem()
	case "set_active_item":
		itemID, ok := intArgument(params.Arguments, "item_id")
		if !ok {
			err = errors.New("item_id must be an integer")
		} else {
			result, err = setActiveItem(itemID)
		}
	case "search_items":
		query, ok := stringArgument(params.Arguments, "query")
		if !ok {
			err = errors.New("query must be a string")
		} else {
			result, err = searchItems(query)
		}
	case "get_item":
		itemID, ok := intArgument(params.Arguments, "item_id")
		if !ok {
			err = errors.New("item_id must be an integer")
		} else {
			result, err = getItem(itemID)
		}
	case "set_workstage":
		workstage, ok := stringArgument(params.Arguments, "workstage")
		if !ok {
			err = errors.New("workstage must be a string")
		} else {
			result, err = setWorkstage(params.Arguments, workstage)
		}
	case "create_item":
		title, ok := stringArgument(params.Arguments, "title")
		if !ok {
			err = errors.New("title must be a string")
		} else {
			result, err = createItem(params.Arguments, title)
		}
	case "clear_active_item":
		err = state.Clear()
		if err == nil {
			result = map[string]any{"cleared": true}
		}
	default:
		err = fmt.Errorf("unknown tool: %s", params.Name)
	}

	if err != nil {
		return textToolError(err.Error())
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return textToolError(err.Error())
	}
	return toolResult{Content: []content{{Type: "text", Text: string(data)}}}
}

func textToolError(message string) toolResult {
	return toolResult{
		Content: []content{{Type: "text", Text: message}},
		IsError: true,
	}
}

func newClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.APIBase == "" {
		return nil, errors.New("no API base URL (run `pwf init`)")
	}
	raw, err := keyring.GetTokens()
	if err != nil {
		return nil, err
	}
	tokens, err := auth.UnmarshalTokens(raw)
	if err != nil {
		return nil, fmt.Errorf("parse stored tokens: %w", err)
	}
	if tokens.IsExpired() && tokens.RefreshToken != "" {
		if cfg.OAuthClientID == "" {
			return nil, errors.New("access token expired but no OAuth client configured (run `pwf init`)")
		}
		refreshed, err := auth.Refresh(cfg.OAuthClientID, cfg.OAuthClientSecret, tokens.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("refresh token: %w", err)
		}
		raw, err = refreshed.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal refreshed tokens: %w", err)
		}
		if err := keyring.SetTokens(raw); err != nil {
			return nil, fmt.Errorf("store refreshed tokens: %w", err)
		}
		tokens = refreshed
	}
	return api.New(cfg.APIBase, tokens.AccessToken), nil
}

func listItems(all bool) ([]api.Item, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	if all {
		return client.ActiveItems()
	}
	return client.MyItems()
}

func getActiveItem() (*api.ItemDetail, error) {
	active, err := state.Load()
	if err != nil {
		return nil, err
	}
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	return client.GetItem(active.ID)
}

func setActiveItem(itemID int) (*state.ActiveItem, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	detail, err := client.GetItem(itemID)
	if err != nil {
		return nil, err
	}
	active := &state.ActiveItem{
		ID: detail.ID, Title: detail.Name, Code: detail.Code,
		UniqueToken: detail.UniqueToken, ProjectID: detail.ProjectID,
		ProjectName: detail.ProjectTitle, PhaseName: detail.PhaseName,
		PhaseID: detail.ItemCollectionID,
	}
	if err := state.Save(active); err != nil {
		return nil, err
	}
	return active, nil
}

func searchItems(query string) ([]api.Item, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	return client.SearchItems(query)
}

func getItem(itemID int) (*api.ItemDetail, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	return client.GetItem(itemID)
}

func setWorkstage(arguments map[string]any, query string) (map[string]any, error) {
	workstages, err := config.LoadWorkstages()
	if err != nil {
		return nil, err
	}
	id, name, ok := workstages.FuzzyMatch(query)
	if !ok {
		return nil, fmt.Errorf("no workstage matching %q", query)
	}
	itemID, hasItemID := intArgument(arguments, "item_id")
	if !hasItemID {
		active, err := state.Load()
		if err != nil {
			return nil, err
		}
		itemID = active.ID
	}
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	if err := client.SetWorkstage(itemID, id); err != nil {
		return nil, err
	}
	return map[string]any{"item_id": itemID, "workstage_id": id, "workstage": name}, nil
}

func createItem(arguments map[string]any, title string) (*api.ItemDetail, error) {
	active, err := state.Load()
	if err != nil {
		return nil, err
	}
	projectID, hasProjectID := intArgument(arguments, "project_id")
	if !hasProjectID {
		projectID = active.ProjectID
	}
	phaseID, hasPhaseID := intArgument(arguments, "phase_id")
	if !hasPhaseID {
		phaseID = active.PhaseID
	}

	client, err := newClient()
	if err != nil {
		return nil, err
	}
	if phaseName, ok := stringArgument(arguments, "phase_name"); ok {
		phases, err := client.GetPhases(projectID)
		if err != nil {
			return nil, fmt.Errorf("fetch phases: %w", err)
		}
		phase, ok := fuzzyPhase(phases, phaseName)
		if !ok {
			return nil, fmt.Errorf("no phase matching %q", phaseName)
		}
		phaseID = phase.ID
	}

	me, err := client.Me()
	if err != nil {
		return nil, fmt.Errorf("get my contact: %w", err)
	}
	contactID, ok := me["id"].(float64)
	if !ok {
		return nil, errors.New("unexpected contact ID type")
	}
	created, err := client.CreateItem("project", projectID, phaseID, title)
	if err != nil {
		return nil, err
	}
	if err := client.AssignContact(created.ID, int(contactID)); err != nil {
		return nil, err
	}
	return client.GetItem(created.ID)
}

func fuzzyPhase(phases []api.Phase, query string) (*api.Phase, bool) {
	q := strings.ToLower(query)
	for i := range phases {
		if strings.ToLower(phases[i].Name) == q {
			return &phases[i], true
		}
	}
	for i := range phases {
		if strings.HasPrefix(strings.ToLower(phases[i].Name), q) {
			return &phases[i], true
		}
	}
	for i := range phases {
		if strings.Contains(strings.ToLower(phases[i].Name), q) {
			return &phases[i], true
		}
	}
	return nil, false
}

func intArgument(arguments map[string]any, key string) (int, bool) {
	value, exists := arguments[key]
	if !exists {
		return 0, false
	}
	number, ok := value.(float64)
	if !ok || number != float64(int(number)) {
		return 0, false
	}
	return int(number), true
}

func stringArgument(arguments map[string]any, key string) (string, bool) {
	value, exists := arguments[key]
	if !exists {
		return "", false
	}
	text, ok := value.(string)
	return text, ok && text != ""
}
