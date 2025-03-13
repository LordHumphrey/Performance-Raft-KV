// Package main provides a simple command-line tool to test the etcd API implementation.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	endpoints = []string{"localhost:2379"}
	command   string
	key       string
	value     string
)

func init() {
	flag.StringVar(&command, "cmd", "get", "Command to execute: get, put, delete")
	flag.StringVar(&key, "key", "", "Key to operate on")
	flag.StringVar(&value, "value", "", "Value to set (only for put command)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -cmd put -key foo -value bar\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -cmd get -key foo\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -cmd delete -key foo\n", os.Args[0])
	}
}

func main() {
	flag.Parse()

	if key == "" {
		fmt.Fprintf(os.Stderr, "Key is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if command == "put" && value == "" {
		fmt.Fprintf(os.Stderr, "Value is required for put command\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create a new etcd client
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create etcd client: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch command {
	case "get":
		resp, err := cli.Get(ctx, key)
		if err != nil {
			log.Fatalf("Failed to get key: %v", err)
		}
		if len(resp.Kvs) == 0 {
			fmt.Printf("Key '%s' not found\n", key)
		} else {
			for _, ev := range resp.Kvs {
				fmt.Printf("Key: %s, Value: %s\n", ev.Key, ev.Value)
			}
		}

	case "put":
		_, err := cli.Put(ctx, key, value)
		if err != nil {
			log.Fatalf("Failed to put key: %v", err)
		}
		fmt.Printf("Successfully put key '%s' with value '%s'\n", key, value)

	case "delete":
		resp, err := cli.Delete(ctx, key)
		if err != nil {
			log.Fatalf("Failed to delete key: %v", err)
		}
		fmt.Printf("Deleted %d key(s)\n", resp.Deleted)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}
