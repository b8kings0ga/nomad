package client

import (
	"github.com/hashicorp/nomad/client/servers"
	"testing"
)

func TestAppendBootstrapServersKeepsAlternateAddress(t *testing.T) {
	a, err := resolveServer("127.0.0.1:14647")
	if err != nil {
		t.Fatal(err)
	}
	got := appendBootstrapServers([]*servers.Server{{Addr: a}}, []string{"127.0.0.1:14647", "[200::1]:14647", "[200::1]:14647", "[invalid"})
	if len(got) != 2 || got[0].Addr.String() != "127.0.0.1:14647" || got[1].Addr.String() != "[200::1]:14647" {
		t.Fatalf("unexpected recovery candidates: %v", got)
	}
	// Repeated advertisements cannot amplify the configured set.
	if next := appendBootstrapServers(got, []string{"[200::1]:14647"}); len(next) != 2 {
		t.Fatal(next)
	}
}
