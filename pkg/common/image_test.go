package common

import (
	"encoding/json"
	"testing"
)

func TestImageJSON(t *testing.T) {
	jsonString := `{"Name":"docker.io/mfenwickbd/perceptor","Sha":"04bb619150cd99cfb21e76429c7a5c2f4545775b07456cb6b9c866c8aff9f9e5","DockerImage":"docker.io/mfenwickbd/perceptor:latest"}`
	var image Image
	err := json.Unmarshal([]byte(jsonString), &image)
	if err != nil {
		t.Errorf("unable to parse %s as JSON: %v", jsonString, err)
		t.Fail()
		panic("a")
	}
	expectedName := "docker.io/mfenwickbd/perceptor"
	if image.Name != expectedName {
		t.Errorf("expected name of %s, got %s", expectedName, image.Name)
	}
}
