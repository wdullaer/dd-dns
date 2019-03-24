package store

import (
	"encoding/json"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/wdullaer/dd-dns/dns"
	"github.com/wdullaer/dd-dns/stringslice"
	"github.com/wdullaer/dd-dns/types"
	"go.uber.org/zap"
)

const bucketName = "dns-mapping"

// BoltDBStore implements the Store interface using a persistant BoltDB instance
type BoltDBStore struct {
	db     *bolt.DB
	logger *zap.SugaredLogger
}

// NewBoltDBStore creates a BoltDBStore persisting its state in the given directory
func NewBoltDBStore(logger *zap.SugaredLogger, dataDir string) (*BoltDBStore, error) {
	db, err := bolt.Open(filepath.Join(dataDir, "dd-dns.db"), 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &BoltDBStore{db: db, logger: logger.Named("boltdb-store")}, nil
}

// CleanUp ensures any pending writes are flushed to disk
// It should be called before closing the program to ensure there is no dataloss
func (store *BoltDBStore) CleanUp() {
	store.logger.Info("Close boltdb connection")
	store.db.Close()
}

// InsertMapping registers that the ContainerID of the DNSMapping supports an A record
// In case the A record is not present in the current state, the callback will be executed
// which should create it at the DNSProvider
// TODO: maybe pass a dns.Provider, rather than a generic callback
func (store *BoltDBStore) InsertMapping(dnsMapping *types.DNSMapping, insertCB func(*types.DNSMapping) error) error {
	return store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		rawRecord := bucket.Get(dnsMapping.GetKey())

		// New record, save it in db and create in dns provider
		if rawRecord == nil {
			payload, err := json.Marshal(types.DNSContainerList{
				Name:          dnsMapping.Name,
				IP:            dnsMapping.IP,
				ContainerList: []string{dnsMapping.ContainerID},
			})
			if err != nil {
				return err
			}
			err = bucket.Put(dnsMapping.GetKey(), payload)
			if err != nil {
				return err
			}
			// Not sure if it's a good idea to keep this IO in the transaction
			// It does guarantee consistency this way
			return insertCB(dnsMapping)
		}
		// Record exists, append containerID
		record := &types.DNSContainerList{}
		err := json.Unmarshal(rawRecord, record)
		if err != nil {
			return err
		}
		if !stringslice.Contains(record.ContainerList, dnsMapping.ContainerID) {
			record.ContainerList = append(record.ContainerList, dnsMapping.ContainerID)
			payload, err := json.Marshal(record)
			if err != nil {
				return err
			}
			err = bucket.Put(dnsMapping.GetKey(), payload)
			if err != nil {
				return err
			}
		}
		// Record exists, containerID is present
		return nil
	})
}

// RemoveMapping removes the ContainerID from the list backing the A record
// In case this was the last ContainerID in the list, the callback will be executed
// to remove the A record from the DNSProvider
func (store *BoltDBStore) RemoveMapping(dnsMapping *types.DNSMapping, removeCB func(*types.DNSMapping) error) error {
	return store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		rawRecord := bucket.Get(dnsMapping.GetKey())

		if rawRecord == nil {
			store.logger.Warn("BoltDBStore - Tried to remove a mapping that was not present in the store")
			return nil
		}

		record := &types.DNSContainerList{}
		err := json.Unmarshal(rawRecord, record)
		if err != nil {
			return err
		}
		record.ContainerList = stringslice.RemoveFirst(record.ContainerList, dnsMapping.ContainerID)

		// No mappings anymore, remove from dns provider
		if len(record.ContainerList) == 0 {
			err := removeCB(dnsMapping)
			if err != nil {
				return err
			}
			return bucket.Delete(dnsMapping.GetKey())
		}

		// Still mappings left, just update store
		payload, err := json.Marshal(record)
		if err != nil {
			return err
		}
		return bucket.Put(dnsMapping.GetKey(), payload)
	})
}

// ReplaceMappings will replace the current list of DNSMappings with the supplied list
// It will interact with the dns.Provider to ensure the remote state is in sync
// It will perform a diff with the current state to minimize the amount of API calls to the dns.Provider
func (store *BoltDBStore) ReplaceMappings(mappings []*types.DNSMapping, provider dns.Provider) error {
	missingItems := []*types.DNSMapping{}
	err := store.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		cursor := bucket.Cursor()

		containerIDs := make([]string, len(mappings))
		for i, mapping := range mappings {
			containerIDs[i] = mapping.ContainerID
		}

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			dnsContainerList := &types.DNSContainerList{}
			json.Unmarshal(v, dnsContainerList)
			store.logger.Infow("Current Mapping", "mapping", dnsContainerList)
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
		return nil
	})
	if err != nil {
		return err
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
