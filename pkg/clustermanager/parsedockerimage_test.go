package clustermanager

import (
	"testing"
)

func TestParseImageID(t *testing.T) {
	name, sha, err := ParseImageIDString("docker-pullable://registry.kipp.blackducksoftware.com/blackducksoftware/hub-registration@sha256:cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043")
	//	name, sha, err := ParseImageIDString("docker-pullable://r.k/h@sha256:cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043")
	if err != nil {
		t.Errorf("expected no error, found %s", err.Error())
		t.Fail()
	}
	if name != "registry.kipp.blackducksoftware.com/blackducksoftware/hub-registration" {
		t.Errorf("incorrect name, got %s", name)
		t.Fail()
	}
	if sha != "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043" {
		t.Errorf("incorrect sha, got %s", sha)
		t.Fail()
	}
}

func TestParseImageIDFail(t *testing.T) {
	name, tag, err := ParseImageIDString("abc")
	if err == nil {
		t.Errorf("expected error, found nil")
		t.Fail()
	}
	if err.Error() != "could not find prefix <docker-pullable://> in <abc>" {
		t.Errorf("incorrect error message: %s", err.Error())
		t.Fail()
	}
	if name != "" {
		t.Errorf("incorrect name: %s", name)
		t.Fail()
	}
	if tag != "" {
		t.Errorf("incorrect tag %s", tag)
		t.Fail()
	}
}

func TestParseImageIDFailMissingSha(t *testing.T) {
	name, tag, err := ParseImageIDString("docker-pullable://abc")
	if err == nil {
		t.Errorf("expected error, found nil")
		t.Fail()
	}
	if err.Error() != "unable to match digestRegexp regex <@sha256:([a-zA-Z0-9]+)$> to input <abc>" {
		t.Errorf("incorrect error message: %s", err.Error())
		t.Fail()
	}
	if name != "" {
		t.Errorf("incorrect name: %s", name)
		t.Fail()
	}
	if tag != "" {
		t.Errorf("incorrect tag %s", tag)
		t.Fail()
	}
}
