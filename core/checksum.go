package core

import (
	"encoding/binary"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"go/types"
)

type CheckSumer interface {
	CreatBucket(string) error
	Diff(string, uint32) (bool, error)
	Save(string, uint32) error
}

type boltdb struct {
	db         *bolt.DB
	bucketName string // current bucket name
}

func NewBolt(db *bolt.DB) CheckSumer {
	return &boltdb{db: db}
}

func (b *boltdb) Bucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte(b.bucketName))
}

func (b *boltdb) CreatBucket(domain string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		var err error
		_, err = tx.CreateBucketIfNotExists([]byte(domain))
		if err != nil {
			return fmt.Errorf("create bucket failed: %s", err)
		}
		b.bucketName = domain
		return nil
	})
}

// imageName是镜像名带tag remoteSum
func (b *boltdb) Diff(imageName string, remoteSum uint32) (bool, error) {
	var (
		diff bool
		err  error
	)

	err = b.db.Batch(func(tx *bolt.Tx) error {
		DBImgBytes := b.Bucket(tx).Get([]byte(imageName))
		if len(DBImgBytes) != int(types.Uint32) { //没读到数据或者长度不对,不能使用binary的方法转uint32，否则会out of range
			diff = true
			return nil
		}
		//和下面的Save同时使用小端或者大端
		// 不同则true
		if remoteSum != binary.LittleEndian.Uint32(DBImgBytes) {
			diff = true
		}
		return nil
	})

	return diff, err
}

func (b *boltdb) Save(imageName string, checkSum uint32) error {
	dstBytesBuf := make([]byte, types.Uint32)
	binary.LittleEndian.PutUint32(dstBytesBuf, checkSum)
	return b.db.Update(func(tx *bolt.Tx) error {
		return b.Bucket(tx).Put([]byte(imageName), dstBytesBuf)
	})
}
