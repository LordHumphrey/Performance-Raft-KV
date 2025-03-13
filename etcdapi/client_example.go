// This file contains an example of how to use the etcd v3 API with our implementation.
// It is not part of the main application, but serves as documentation and testing.
package etcdapi

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ClientExample shows how to use the etcd v3 client with our implementation.
func ClientExample() {
	// Create a new etcd client
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	// Set a key
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Put(ctx, "foo", "bar")
	cancel()
	if err != nil {
		log.Fatalf("failed to put key: %v", err)
	}
	fmt.Println("Put key 'foo' with value 'bar'")

	// Get the key
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := cli.Get(ctx, "foo")
	cancel()
	if err != nil {
		log.Fatalf("failed to get key: %v", err)
	}
	for _, ev := range resp.Kvs {
		fmt.Printf("Key: %s, Value: %s\n", ev.Key, ev.Value)
	}

	// Delete the key
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	_, err = cli.Delete(ctx, "foo")
	cancel()
	if err != nil {
		log.Fatalf("failed to delete key: %v", err)
	}
	fmt.Println("Deleted key 'foo'")

	// Verify the key is deleted
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	resp, err = cli.Get(ctx, "foo")
	cancel()
	if err != nil {
		log.Fatalf("failed to get key: %v", err)
	}
	if len(resp.Kvs) == 0 {
		fmt.Println("Key 'foo' was successfully deleted")
	} else {
		fmt.Println("Key 'foo' still exists!")
	}
}
