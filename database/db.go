package database

import (
	"fmt"
	"strconv"

	bolt "go.etcd.io/bbolt"
)

const (
	bucketResultsGuids = "ResultsGuidsCache"
	bucketSubscribers  = "Subscribers"
)

type DbAdapter struct {
	db *bolt.DB
}

func Open(path string) (*DbAdapter, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("открытие БД: %w", err)
	}

	adapter := &DbAdapter{db: db}
	if err := adapter.initBuckets(); err != nil {
		db.Close()
		return nil, err
	}

	return adapter, nil
}

func (d *DbAdapter) initBuckets() error {
	return d.db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketResultsGuids)); err != nil {
			return fmt.Errorf("создание bucket %s: %w", bucketResultsGuids, err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketSubscribers)); err != nil {
			return fmt.Errorf("создание bucket %s: %w", bucketSubscribers, err)
		}
		return nil
	})
}

func (d *DbAdapter) Close() error {
	return d.db.Close()
}

func (d *DbAdapter) HasResult(id string) (bool, error) {
	var found bool
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketResultsGuids))
		found = b.Get([]byte(id)) != nil
		return nil
	})
	return found, err
}

func (d *DbAdapter) MarkResult(id string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketResultsGuids))
		return b.Put([]byte(id), []byte{})
	})
}

func (d *DbAdapter) AddSubscriber(chatID int64) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSubscribers))
		key := []byte(strconv.FormatInt(chatID, 10))
		return b.Put(key, []byte{})
	})
}

func (d *DbAdapter) ListSubscribers() ([]int64, error) {
	var ids []int64
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSubscribers))
		return b.ForEach(func(k, _ []byte) error {
			id, err := strconv.ParseInt(string(k), 10, 64)
			if err != nil {
				return fmt.Errorf("некорректный chat id %q: %w", k, err)
			}
			ids = append(ids, id)
			return nil
		})
	})
	return ids, err
}
