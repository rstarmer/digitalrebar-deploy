package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/hashicorp/consul/api"
	"github.com/kmanley/go-http-auth"
)

/* Of the form:
{
  "rebar": {
    "realm": "Rebar",
    "capabilities": [],
    "password": "rebar1",
    "digestpassword": ""
  }
}
*/

type User struct {
	Realm          string
	Password       string
	Digestpassword string
	Capabilities   []string
}

type JsonData map[string]User

type JsonFile struct {
	auth.File
	Data JsonData
}

func reload_jsonfile(j *JsonFile) {
	data, err := ioutil.ReadFile(j.Path)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &j.Data)
	if err != nil {
		panic(err)
	}
}

func get_password(user, realm string, data JsonData, pwdtype string) string {
	u_data, exists := data[user]
	if !exists {
		log.Printf("Failed login attempt - realm: %v no user: %v", realm, user)
		return ""
	}
	if pwdtype == "basic" {
		log.Printf("Basic login attempt - realm: %v user: %v", realm, user)
		return u_data.Password
	}
	log.Printf("Digest login attempt - realm: %v user: %v", realm, user)
	return u_data.Digestpassword
}

func get_capabilities(user, realm string, data JsonData) []string {
	u_data, exists := data[user]
	if !exists {
		return []string{}
	}
	if len(u_data.Capabilities) == 0 {
		return []string{"Read-Only"}
	}
	return u_data.Capabilities
}

type CapabilityProvider func(user, realm string) []string

/*
  SecretProvider implementation based on json files.
  reload file on changes. Will panic on syntax errors in
  files.
*/
func JsonFileProvider(filename string, pwdtype string) (auth.SecretProvider, CapabilityProvider, error) {
	j := &JsonFile{File: auth.File{Path: filename}}
	j.Reload = func() { reload_jsonfile(j) }
	return func(user, realm string) string {
			j.ReloadIfNeeded()
			return get_password(user, realm, j.Data, pwdtype)
		},
		func(user, realm string) []string {
			j.ReloadIfNeeded()
			return get_capabilities(user, realm, j.Data)
		},
		nil
}

type ConsulFile struct {
	kv         *api.KV
	backingKey string
	Data       JsonData
}

func JsonConsulProvider(key, dbInit string, pwdtype string) (auth.SecretProvider, CapabilityProvider, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, nil, err
	}
	if _, err := client.Agent().Self(); err != nil {
		return nil, nil, err
	}

	store := client.KV()
	pair, _, err := store.Get(key, nil)
	if err != nil {
		return nil, nil, err
	}
	if pair == nil {
		data, err := ioutil.ReadFile(dbInit)
		if err != nil {
			return nil, nil, err
		}
		_, err = store.Put(&api.KVPair{Key: key, Value: data}, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	cf := &ConsulFile{kv: store, backingKey: key}
	pair, _, err = store.Get(key, nil)
	if err != nil {
		return nil, nil, err
	}
	err = json.Unmarshal(pair.Value, &cf.Data)
	if err != nil {
		return nil, nil, err
	}

	// GREG: Start consul key watcher with lock to update json data

	return func(user, realm string) string {
			// GREG: Add lock
			return get_password(user, realm, cf.Data, pwdtype)
		},
		func(user, realm string) []string {
			return get_capabilities(user, realm, cf.Data)
		},
		nil

}