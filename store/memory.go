package store

import (
	"log"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/wdullaer/docker-dns-updater/dns"
	"github.com/wdullaer/docker-dns-updater/stringslice"
	"github.com/wdullaer/docker-dns-updater/types"
)

const tableName = "dns-mapping"

type MemoryStore struct {
	db *memdb.MemDB
}

func NewMemoryStore() (*MemoryStore, error) {
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
	return &MemoryStore{db: db}, nil
}

func (*MemoryStore) CleanUp() {}

func (store *MemoryStore) InsertMapping(mapping *types.DNSMapping, cb func(string, string) error) error {
	txn := store.db.Txn(true)
	defer txn.Abort()

	rawRecord, err := txn.First(tableName, "id", mapping.Name, mapping.IP)
	if err != nil {
		return err
	}

	if rawRecord == nil {
		log.Println("[INFO] Insert record into DNS")
		if err = cb(mapping.Name, mapping.IP); err != nil {
			return err
		}
		err = txn.Insert(tableName, &types.DNSLabel{
			Name:        mapping.Name,
			IP:          mapping.IP,
			ContainerID: []string{mapping.ContainerID},
		})
		if err != nil {
			return err
		}
		txn.Commit()
		return nil
	}

	record := rawRecord.(*types.DNSLabel)

	if !stringslice.Contains(record.ContainerID, mapping.ContainerID) {
		if err = txn.Delete(tableName, record); err != nil {
			return err
		}
		record.ContainerID = append(record.ContainerID, mapping.ContainerID)
		if err = txn.Insert(tableName, record); err != nil {
			return err
		}
	}

	txn.Commit()
	return nil
}

func (store *MemoryStore) RemoveMapping(mapping *types.DNSMapping, cb func(string, string) error) error {
	txn := store.db.Txn(true)
	defer txn.Abort()

	rawRecord, err := txn.First(tableName, "containerid", mapping.ContainerID)
	if err != nil {
		return err
	}
	if rawRecord == nil {
		log.Printf("[WARN] Trying to remove non-existing DNS-container mapping. (containerID: %s)", mapping.ContainerID)
		return nil
	}

	if err = txn.Delete(tableName, rawRecord); err != nil {
		return err
	}

	record := rawRecord.(*types.DNSLabel)
	record.ContainerID = stringslice.RemoveFirst(record.ContainerID, mapping.ContainerID)

	if len(record.ContainerID) == 0 {
		if err = cb(mapping.Name, mapping.IP); err != nil {
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

func (store *MemoryStore) ReplaceMappings(mappings []*types.DNSMapping, provider dns.DNSProvider) error {
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
		dnsLabel := item.(*types.DNSLabel)
		for _, containerID := range dnsLabel.ContainerID {
			if stringslice.Contains(containerIDs, containerID) {
				break
			}
			missingItems = append(missingItems, &types.DNSMapping{
				Name:        dnsLabel.Name,
				IP:          dnsLabel.IP,
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
