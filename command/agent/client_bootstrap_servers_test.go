package agent

import (
	"reflect"
	"testing"
)

func TestStaticClientBootstrapServersBeforeRestore(t *testing.T) {
	existing := []string{"server.example:14647"}
	got := appendStaticClientBootstrapServers(existing, []string{"[200::1]:14647", "192.0.2.1:14647", "server.example:14647", "provider=aws tag_key=nomad", "bad[", ""})
	want := []string{"server.example:14647", "[200::1]:14647", "192.0.2.1:14647"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	got[0] = "changed"
	if existing[0] != "server.example:14647" {
		t.Fatal("mutated caller config")
	}
}

func TestConvertClientConfigPrimesRetryJoinBeforeClientConstruction(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Client.ServerJoin.RetryJoin = []string{"[200::1]:14647", "192.0.2.1:14647"}
	client, err := convertClientConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(client.Servers, cfg.Client.ServerJoin.RetryJoin) {
		t.Fatalf("restore would start without bootstrap servers: %v", client.Servers)
	}
}
