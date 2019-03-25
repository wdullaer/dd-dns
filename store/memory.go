package store

import (
	memdb "github.com/hashicorp/go-memdb"
	"github.com/wdullaer/dd-dns/dns"
	"github.com/wdullaer/dd-dns/stringslice"
	"github.com/wdullaer/dd-dns/types"
	"go.uber.org/zap"
)

const tableName = "dns-mapping"

// MemoryStore implements the Store interface using an ephemeral in memory database
type MemoryStore struct {
	db     *memdb.MemDB
	logger *zap.SugaredLogger
}

// NewMemoryStore returns a new instance of a MemoryStore
func NewMemoryStore(logger *zap.SugaredLogger) (*MemoryStore, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			tableName: &memdb.TableSchema{
				Name: tableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.CompoundIndex{Indexes: []memdb.Indexer{&memdb.StringFieldIndex{Field: "Name"}, &memdb.StringFieldIndex{Field: "IP"}}},
					},
					"containerid": &memdb.IndexSchema{
						Name:    "containerid",
						Unique:  true,
						Indexer: &memdb.StringSliceFieldIndex{Field: "ContainerID"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}
	return &MemoryStore{db: db, logger: logger.Named("memory-store")}, nil
}

// CleanUp is a no-op for the MemoryStore
func (*MemoryStore) CleanUp() {}

// InsertMapping registers that the ContainerID of the DNSMapping supports an A record
// In case the A record is not present in the current state, the callback will be executed
// which should create it at the DNSProvider
// TODO: maybe pass a dns.Provider, rather than a generic callback
func (store *MemoryStore) InsertMapping(mapping *types.DNSMapping, cb func(*types.DNSMapping) error) error {
	txn := store.db.Txn(true)
	defer txn.Abort()

	rawRecord, err := txn.First(tableName, "id", mapping.Name, mapping.IP.String())
	if err != nil {
		return err
	}

	if rawRecord == nil {
		if err = cb(mapping); err != nil {
			return err
		}
		err = txn.Insert(tableName, &types.DNSContainerList{
			Name:          mapping.Name,
			IP:            mapping.IP,
			ContainerList: []string{mapping.ContainerID},
		})
		if err != nil {
			return err
		}
		txn.Commit()
		return nil
	}

	record := rawRecord.(*types.DNSContainerList)

	if !stringslice.Contains(record.ContainerList, mapping.ContainerID) {
		if err = txn.Delete(tableName, record); err != nil {
			return err
		}
		record.ContainerList = append(record.ContainerList, mapping.ContainerID)
		if err = txn.Insert(tableName, record); err != nil {
			return err
		}
	}

	txn.Commit()
	return nil
}

// RemoveMapping removes the ContainerID from the list backing the A record
// In case this was the last ContainerID in the list, the callback will be executed
// to remove the A record from the DNSProvider
func (store *MemoryStore) RemoveMapping(mapping *types.DNSMapping, cb func(*types.DNSMapping) error) error {
	txn := store.db.Txn(true)
	defer txn.Abort()

	rawRecord, err := txn.First(tableName, "containerid", mapping.ContainerID)
	if err != nil {
		return err
	}
	if rawRecord == nil {
		store.logger.Warnw("Trying to remove non-existing DNS-container mapping. (containerID: %s)", "containerID", mapping.ContainerID)
		return nil
	}

	if err = txn.Delete(tableName, rawRecord); err != nil {
		return err
	}

	record := rawRecord.(*types.DNSContainerList)
	record.ContainerList = stringslice.RemoveFirst(record.ContainerList, mapping.ContainerID)

	if len(record.ContainerList) == 0 {
		if err = cb(mapping); err != nil {
			return err
		}
	} else {
		if err = txn.Insert(tableName, record); err != nil {
			return err
		}
	}

	txn.Commit()
	return nil
}

// ReplaceMappings will replace the current list of DNSMappings with the supplied list
// It will interact with the dns.Provider to ensure the remote state is in sync
// It will perform a diff with the current state to minimize the amount of API calls to the dns.Provider
func (store *MemoryStore) ReplaceMappings(mappings []*types.DNSMapping, provider dns.Provider) error {
	txn := store.db.Txn(false)
	defer txn.Abort()

	containerIDs := make([]string, len(mappings))
	for i, mapping := range mappings {
		containerIDs[i] = mapping.ContainerID
	}

	iterator, err := txn.Get(tableName, "containerid")
	if err != nil {
		return err
	}

	missingItems := []*types.DNSMapping{}
	for item := iterator.Next(); item != nil; item = iterator.Next() {
		dnsContainerList := item.(*types.DNSContainerList)
		for _, containerID := range dnsContainerList.ContainerList {
			if stringslice.Contains(containerIDs, containerID) {
				break
			}
			missingItems = append(missingItems, &types.DNSMapping{
				Name:        dnsContainerList.Name,
				IP:          dnsContainerList.IP,
				ContainerID: containerID,
			})
		}
	}

	for i := range mappings {
		err := store.InsertMapping(mappings[i], provider.AddHostnameMapping)
		if err != nil {
			return err
		}
	}

	for i := range missingItems {
		err := store.RemoveMapping(missingItems[i], provider.RemoveHostnameMapping)
		if err != nil {
			return err
		}
	}

	return nil
}
