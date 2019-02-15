package store

import (
	"encoding/json"
	"log"

	"github.com/boltdb/bolt"
	"github.com/wdullaer/docker-dns-updater/dns"
	"github.com/wdullaer/docker-dns-updater/stringslice"
	"github.com/wdullaer/docker-dns-updater/types"
)

const bucketName = "dns-mapping"

type BoltDBStore struct {
	db *bolt.DB
}

func NewBoltDBStore() (*BoltDBStore, error) {
	db, err := bolt.Open("docker-dns-updater.db", 0600, nil)
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

	return &BoltDBStore{db: db}, nil
}

func (store *BoltDBStore) CleanUp() {
	log.Printf("[INFO] Close boltdb connection")
	store.db.Close()
}

func (store *BoltDBStore) InsertMapping(dnsMapping *types.DNSMapping, insertCB func(string, string) error) error {
	return store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		rawRecord := bucket.Get(dnsMapping.GetKey())

		// New record, save it in db and create in dns provider
		if rawRecord == nil {
			payload, err := json.Marshal(types.DNSLabel{
				Name:        dnsMapping.Name,
				IP:          dnsMapping.IP,
				ContainerID: []string{dnsMapping.ContainerID},
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
			return insertCB(dnsMapping.Name, dnsMapping.IP)
		}
		// Record exists, append containerID
		record := &types.DNSLabel{}
		err := json.Unmarshal(rawRecord, record)
		if err != nil {
			return err
		}
		if !stringslice.Contains(record.ContainerID, dnsMapping.ContainerID) {
			record.ContainerID = append(record.ContainerID, dnsMapping.ContainerID)
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

func (store *BoltDBStore) RemoveMapping(dnsMapping *types.DNSMapping, removeCB func(string, string) error) error {
	return store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		rawRecord := bucket.Get(dnsMapping.GetKey())

		if rawRecord == nil {
			log.Println("[WARN] BoltDBStore - Tried to remove a mapping that was not present in the store")
			return nil
		}

		record := &types.DNSLabel{}
		err := json.Unmarshal(rawRecord, record)
		if err != nil {
			return err
		}
		record.ContainerID = stringslice.RemoveFirst(record.ContainerID, dnsMapping.ContainerID)

		// No mappings anymore, remove from dns provider
		if len(record.ContainerID) == 0 {
			err := removeCB(dnsMapping.Name, dnsMapping.IP)
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

func (store *BoltDBStore) ReplaceMappings(mappings []*types.DNSMapping, provider dns.DNSProvider) error {
	missingItems := []*types.DNSMapping{}
	err := store.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		cursor := bucket.Cursor()

		containerIDs := make([]string, len(mappings))
		for i, mapping := range mappings {
			containerIDs[i] = mapping.ContainerID
		}

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			dnsLabel := &types.DNSLabel{}
			json.Unmarshal(v, dnsLabel)
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
