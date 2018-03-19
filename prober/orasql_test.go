package prober

import (
	"fmt"
	"testing"
)

func TestLoadDBConfig(t *testing.T) {
	var dbc *DBConnConfig = &DBConnConfig{}
	dbc.loadDBConfig("testdata/conns.yml")
	t.Log(len(dbc.Conns))
	for k, v := range dbc.Conns {
		t.Log(fmt.Printf("Key:%s Value:%s\n", k, v))
	}
}
