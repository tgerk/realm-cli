package mock

import "github.com/10gen/realm-cli/internal/cloud/atlas"

// AtlasClient is a mocked Atlas client
type AtlasClient struct {
	atlas.Client
	GroupsFn    func() ([]atlas.Group, error)
	ClustersFn  func(groupID string) ([]atlas.Cluster, error)
	DataLakesFn func(groupID string) ([]atlas.DataLake, error)
}

// Groups calls the mocked Groups implementation if provided,
// otherwise the call falls back to the underlying atlas.Client implementation.
// NOTE: this may panic if the underlying atlas.Client is left undefined
func (ac AtlasClient) Groups() ([]atlas.Group, error) {
	if ac.GroupsFn != nil {
		return ac.GroupsFn()
	}
	return ac.Client.Groups()
}

// Clusters calls the mocked Clusters implementation if provided,
// otherwise the call falls back to the underlying atlas.Client implementation.
// NOTE: this may panic if the underlying atlas.Client is left undefined
func (ac AtlasClient) Clusters(groupID string) ([]atlas.Cluster, error) {
	if ac.ClustersFn != nil {
		return ac.ClustersFn(groupID)
	}
	return ac.Client.Clusters(groupID)
}

// DataLakes calls the mocked DataLakes implementation if provided,
// otherwise the call falls back to the underlying atlas.Client implementation.
// NOTE: this may panic if the underlying atlas.Client is left undefined
func (ac AtlasClient) DataLakes(groupID string) ([]atlas.DataLake, error) {
	if ac.DataLakesFn != nil {
		return ac.DataLakesFn(groupID)
	}
	return ac.Client.DataLakes(groupID)
}
