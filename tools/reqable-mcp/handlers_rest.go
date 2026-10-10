package main

import (
	"fmt"
	"reqable-mcp/store"
)

func init() {
	handlers["rest_http_create_from_url"] = handleRestHttpCreateFromUrl
	handlers["rest_http_create_from_curl"] = handleRestHttpCreateFromCurl
	handlers["rest_http_update"] = handleRestHttpUpdate
	handlers["rest_http_get"] = handleRestHttpGet
	handlers["rest_websocket_create_from_url"] = handleRestWebsocketCreateFromUrl
	handlers["rest_websocket_update"] = handleRestWebsocketUpdate
	handlers["rest_websocket_get"] = handleRestWebsocketGet
}

func handleRestHttpCreateFromUrl(args map[string]any) (string, error) {
	name, url, err := reqNameURL(args)
	if err != nil {
		return "", err
	}
	id, err := store.RestHTTPCreateFromURL(ud(args), name, url)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http tab %s staged", id), nil
}

func handleRestHttpCreateFromCurl(args map[string]any) (string, error) {
	name, curl := strArg(args, "name", ""), strArg(args, "curl", "")
	if name == "" || curl == "" {
		return "", badArgs("name and curl are required")
	}
	id, err := store.RestHTTPCreateFromCurl(ud(args), name, curl)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http tab %s staged", id), nil
}

func handleRestHttpUpdate(args map[string]any) (string, error) {
	id := strArg(args, "id", "")
	tab := objArg(args, "tab")
	if id == "" || tab == nil {
		return "", badArgs("id and tab are required")
	}
	if err := store.RestHTTPUpdate(ud(args), id, tab); err != nil {
		return "", err
	}
	return fmt.Sprintf("http tab %s updated", id), nil
}

func handleRestHttpGet(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	tab, err := store.RestHTTPGet(ud(args), id)
	if err != nil {
		return "", err
	}
	return marshalPretty(tab), nil
}

func handleRestWebsocketCreateFromUrl(args map[string]any) (string, error) {
	name, url, err := reqNameURL(args)
	if err != nil {
		return "", err
	}
	id, err := store.RestWSCreateFromURL(ud(args), name, url)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("websocket tab %s staged", id), nil
}

func handleRestWebsocketUpdate(args map[string]any) (string, error) {
	id := strArg(args, "id", "")
	tab := objArg(args, "tab")
	if id == "" || tab == nil {
		return "", badArgs("id and tab are required")
	}
	if err := store.RestWSUpdate(ud(args), id, tab); err != nil {
		return "", err
	}
	return fmt.Sprintf("websocket tab %s updated", id), nil
}

func handleRestWebsocketGet(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	tab, err := store.RestWSGet(ud(args), id)
	if err != nil {
		return "", err
	}
	return marshalPretty(tab), nil
}
