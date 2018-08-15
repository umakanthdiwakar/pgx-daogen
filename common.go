package main

import (
	"encoding/json"
	"io/ioutil"
)

type QueryInfo struct {
	Name  string
	Query string
}

type Genstruct struct {
	Hostname    string
	Dbname      string
	Username    string
	Password    string
	Tables      []string
	Queries     []QueryInfo
	PackageName string
}

func GetGenData(fileName string) *Genstruct {
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}
	v := Genstruct{}
	v.Queries = []QueryInfo{}
	err = json.Unmarshal(bytes, &v)
	if err != nil {
		panic(err)
	}
	return &v
}
