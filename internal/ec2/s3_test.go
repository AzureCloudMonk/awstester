package ec2

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	ec2config "github.com/aws/awstester/internal/ec2/config"
)

func TestS3(t *testing.T) {
	if os.Getenv("RUN_AWS_TESTS") != "1" {
		t.Skip()
	}

	cfg := ec2config.NewDefault()

	ec, err := NewDeployer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	md, ok := ec.(*embedded)
	if !ok {
		t.Fatalf("expected '*embedded', got %v", reflect.TypeOf(ec))
	}

	f, ferr := ioutil.TempFile(os.TempDir(), "test")
	if ferr != nil {
		t.Fatal(ferr)
	}
	_, err = f.Write([]byte("hello world!"))
	if err != nil {
		t.Fatal(err)
	}
	localPath := f.Name()
	f.Close()
	defer os.RemoveAll(localPath)

	if err = md.toS3(localPath, "hello-world"); err != nil {
		t.Fatal(err)
	}
	if err = md.deleteBucket(); err != nil {
		t.Fatal(err)
	}
}
