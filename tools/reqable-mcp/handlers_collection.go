package main

import (
	"errors"
	"fmt"
	"reqable-mcp/store"
)

func init() {
	handlers["collection_list"] = handleCollectionList
	handlers["collection_create"] = handleCollectionCreate
	handlers["collection_get"] = handleCollectionGet
	handlers["collection_delete"] = handleCollectionDelete
	handlers["collection_update"] = handleCollectionUpdate
	handlers["collection_structure"] = handleCollectionStructure
	handlers["collection_folder_create"] = handleCollectionFolderCreate
	handlers["collection_folder_get"] = handleCollectionFolderGet
	handlers["collection_api_create"] = handleCollectionApiCreate
	handlers["collection_api_add"] = handleCollectionApiAdd
	handlers["collection_api_get"] = handleCollectionApiGet
	handlers["collection_api_update"] = handleCollectionApiUpdate
	handlers["collection_api_delete"] = handleCollectionApiDelete
	handlers["collection_api_generate_curl"] = handleCollectionApiGenerateCurl
}

func handleCollectionList(args map[string]any) (string, error) {
	cols, err := store.ListCollections(ud(args))
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(cols))
	for _, c := range cols {
		lines = append(lines, fmt.Sprintf("%v %v", c["id"], c["name"]))
	}
	return listText("collections", len(cols), lines), nil
}

func handleCollectionCreate(args map[string]any) (string, error) {
	name := strArg(args, "name", "")
	if name == "" {
		return "", badArgs("name is required")
	}
	id, err := store.AddCollection(ud(args), name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("collection %s %s -> %s", id, name, restartNote), nil
}

func handleCollectionGet(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	c, err := store.CollectionGet(ud(args), id)
	if err != nil {
		return "", err
	}
	return marshalPretty(c), nil
}

func handleCollectionDelete(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	found, err := store.CollectionDelete(ud(args), id)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("collection " + id + " not found")
	}
	return fmt.Sprintf("collection %s deleted", id), nil
}

func handleCollectionUpdate(args map[string]any) (string, error) {
	id, err := reqID(args)
	if err != nil {
		return "", err
	}
	patch := map[string]any{}
	if hasArg(args, "name") {
		patch["name"] = strArg(args, "name", "")
	}
	if err := store.CollectionUpdate(ud(args), id, patch); err != nil {
		return "", err
	}
	return fmt.Sprintf("collection %s updated -> %s", id, restartNote), nil
}

func handleCollectionStructure(args map[string]any) (string, error) {
	cols, err := store.ListCollections(ud(args))
	if err != nil {
		return "", err
	}
	return marshalPretty(map[string]any{"collections": cols}), nil
}

func handleCollectionFolderCreate(args map[string]any) (string, error) {
	collectionID, name := strArg(args, "collectionId", ""), strArg(args, "name", "")
	if collectionID == "" || name == "" {
		return "", badArgs("collectionId and name are required")
	}
	id, err := store.CollectionFolderCreate(ud(args), collectionID, name, strArg(args, "parentId", ""))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("folder %s %s -> %s", id, name, restartNote), nil
}

func handleCollectionFolderGet(args map[string]any) (string, error) {
	collectionID, id, err := reqColItem(args)
	if err != nil {
		return "", err
	}
	ref, err := store.CollectionFolderGet(ud(args), collectionID, id)
	if err != nil {
		return "", err
	}
	return marshalPretty(ref), nil
}

func handleCollectionApiCreate(args map[string]any) (string, error) {
	collectionID, curl := strArg(args, "collectionId", ""), strArg(args, "curl", "")
	if collectionID == "" || curl == "" {
		return "", badArgs("collectionId and curl are required")
	}
	id, err := store.CollectionAPICreateFromCurl(ud(args), collectionID, curl, strArg(args, "parentId", ""))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("api %s staged -> %s", id, restartNote), nil
}

func handleCollectionApiAdd(args map[string]any) (string, error) {
	collectionID := strArg(args, "collectionId", "")
	api := objArg(args, "api")
	if collectionID == "" || api == nil {
		return "", badArgs("collectionId and api are required")
	}
	id, err := store.CollectionAPIAdd(ud(args), collectionID, api, strArg(args, "parentId", ""))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("api %s added -> %s", id, restartNote), nil
}

func handleCollectionApiGet(args map[string]any) (string, error) {
	collectionID, id, err := reqColItem(args)
	if err != nil {
		return "", err
	}
	ref, err := store.CollectionAPIGet(ud(args), collectionID, id)
	if err != nil {
		return "", err
	}
	return marshalPretty(ref), nil
}

func handleCollectionApiUpdate(args map[string]any) (string, error) {
	collectionID := strArg(args, "collectionId", "")
	api := objArg(args, "api")
	if collectionID == "" || api == nil {
		return "", badArgs("collectionId and api are required")
	}
	if err := store.CollectionAPIUpdate(ud(args), collectionID, api); err != nil {
		return "", err
	}
	return fmt.Sprintf("api updated -> %s", restartNote), nil
}

func handleCollectionApiDelete(args map[string]any) (string, error) {
	collectionID, id, err := reqColItem(args)
	if err != nil {
		return "", err
	}
	found, err := store.CollectionAPIDelete(ud(args), collectionID, id)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("api " + id + " not found")
	}
	return fmt.Sprintf("api %s deleted", id), nil
}

func handleCollectionApiGenerateCurl(args map[string]any) (string, error) {
	collectionID, id, err := reqColItem(args)
	if err != nil {
		return "", err
	}
	curl, err := store.CollectionAPIGenerateCurl(ud(args), collectionID, id)
	if err != nil {
		return "", err
	}
	return curl, nil
}
