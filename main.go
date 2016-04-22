package main

import (
	"bytes"
	"compress/lzw"
	"crypto/md5"
	"crypto/tls"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul-template/dependency"
)

type KeyPair []struct {
	LockIndex   uint64
	Key         string
	Flags       uint64
	Value       []byte
	CreateIndex uint64
	ModifyIndex uint64
}

type templateData struct {
	Data map[string]interface{}
}

type Config struct {
	addr string
	file string
	hash string
}

func getFlags(config *Config) *Config {
	const (
		consulDefault = "localhost:8500"
		consulDesc    = "Consul address"
		fileDefault   = ""
		fileDesc      = "Consul-template file; must end in .ctmpl"
	)

	flag.StringVar(&config.addr, "consul", consulDefault, consulDesc)
	flag.StringVar(&config.file, "file", fileDefault, fileDesc)
	flag.Parse()
	return config
}

func getMd5(config *Config) string {
	contents, err := ioutil.ReadFile(config.file)
	if err != nil {
		log.Fatal(err)
	}

	hash := md5.Sum(contents)
	return hex.EncodeToString(hash[:])
}

func getValue(config *Config) []byte {
	url := fmt.Sprintf(
		"https://%s/v1/kv/consul-template/dedup/%s/data", config.addr, config.hash)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := http.Client{Transport: tr}
	resp, err := c.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != 200 {
		log.Fatal(err)
	}

	var keyPair KeyPair
	if err := json.Unmarshal(body, &keyPair); err != nil {
		log.Fatal(err)
	}

	return keyPair[0].Value
}

func decodeValue(value []byte) {
	r := bytes.NewReader(value)
	decompress := lzw.NewReader(r, lzw.LSB, 8)
	defer decompress.Close()
	dec := gob.NewDecoder(decompress)

	var td templateData
	if err := dec.Decode(&td); err != nil {
		log.Fatal(err)
	}

	// spew.Dump(td.Data)
	for k, v := range td.Data {
		serviceName := strings.Replace(k, "HealthServices|", "", -1)
		fmt.Println(serviceName)
		for _, j := range v.([]*dependency.HealthService) {
			// spew.Dump(j)
			fmt.Printf("  %s\n", j.Node)
		}
		fmt.Println()
	}
}

func main() {
	var config *Config = &Config{}
	config = getFlags(config)

	if len(os.Args) <= 1 {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := filepath.Ext(config.file); err != ".ctmpl" {
		fmt.Println("File extension needs to end in .ctmpl.")
		os.Exit(1)
	}

	config.hash = getMd5(config)
	fmt.Printf("key hash: %s\n\n", config.hash)

	value := getValue(config)
	decodeValue(value)
}
