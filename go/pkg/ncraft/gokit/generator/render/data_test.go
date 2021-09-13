package render

/*
import (
	"os"
	"path/filepath"
	"testing"
)

var gopath []string

func init() {
	gopath = filepath.SplitList(os.Getenv("GOPATH"))
}

func TestNewData(t *testing.T) {
	const def = `
		syntax = "proto3";

		// General package
		package general;

		import "github.com/frankee/ncraft/deftree/googlethirdparty/annotations.proto";

		// RequestMessage is so foo
		message RequestMessage {
			string input = 1;
		}

		// ResponseMessage is so bar
		message ResponseMessage {
			string output = 1;
		}

		// ProtoInterface is a service
		service ProtoInterface {
			// ProtoMethod is simple. Like a gopher.
			rpc ProtoMethod (RequestMessage) returns (ResponseMessage) {
				// No {} in path and no body, everything is in the query
				option (google.api.http) = {
					get: "/route"
				};
			}
		}
	`
	sd, err := svcdef.NewFromString(def, gopath)
	if err != nil {
		t.Fatal(err)
	}

	conf := Config{
		Repository: "github.com/frankee/ncraft/gengokit/general-service",
		PBGoPackage: "github.com/frankee/ncraft/gengokit/general-service",
	}

	te, err := NewData(sd, conf)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := te.PackageName, sd.PkgName; got != want {
		t.Fatalf("\n`%v` was PackageName\n`%v` was wanted", got, want)
	}
}
*/