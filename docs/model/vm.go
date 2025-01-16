package docs

type LabelValue struct {
       Label string `json:"label" example:"name"`
       Value string `json:"value" example:"32482124-6d7d-47a8-b4a9-dea50e50823f"`
}

type ImagesListResponse struct {
       Error bool `json:"error"`
       Images []LabelValue `json:"images"`
}

type FlavorsListResponse struct {
       Error bool `json:"error"`
       Flavors []LabelValue `json:"flavors"`
}

type KeysListResponse struct {
       Error bool `json:"error"`
       Keys []LabelValue `json:"keys"`
}

type SnapshotRequest struct {
        ID string `json:"id" example:"32482124-6d7d-47a8-b4a9-dea50e50823f"`
        Name string `json:"name" example:"ubuntu_snap"`
}

