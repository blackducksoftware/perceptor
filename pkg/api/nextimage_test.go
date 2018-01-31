package api

import (
	"encoding/json"
	"testing"
)

func TestImageJSON(t *testing.T) {
	jsonString := `{"Image":{"Name":"docker.io/mfenwickbd/perceptor","Sha":"04bb619150cd99cfb21e76429c7a5c2f4545775b07456cb6b9c866c8aff9f9e5","DockerImage":"docker.io/mfenwickbd/perceptor:latest"}}`
	// var nextImage NextImage
	// err := json.Unmarshal([]byte(jsonString), &nextImage)
	var nextImage *NextImage
	err := json.Unmarshal([]byte(jsonString), nextImage)
	if err != nil {
		t.Errorf("unable to parse %s as JSON: %v", jsonString, err)
		t.Fail()
		return
	}
	expectedName := "docker.io/mfenwickbd/perceptor"
	if nextImage.Image.Name != expectedName {
		t.Errorf("expected name of %s, got %s", expectedName, nextImage.Image.Name)
	}
}
