package testing

import (
	"Assignment2/firebase"
	"fmt"
	"testing"
	"time"
	"unicode/utf8"
)

const serviceAccountPath = "./demo-service-account.json"
const testCollection = "mess"

// TestInitializeFirestore checks if initializing is successful
func TestInitializeFirestore(t *testing.T) {
	fs := firebase.FirestoreContext{}
	err := fs.Initialize(serviceAccountPath)
	defer fs.Close()
	if err != nil {
		t.Error("could not initialize")
	}
}

// TestAddDocument checks if document can be added to collection
func TestAddDocument(t *testing.T) {
	fs := firebase.FirestoreContext{}
	err := fs.Initialize(serviceAccountPath)
	defer fs.Close()
	if err != nil {
		t.Error("could not initialize")
	}

	testData := map[string]interface{}{
		"first":  "first_value",
		"second": "second_value",
		"third":  "third_value",
	}

	id, err := fs.AddDocument("testCollection", testData)

	if err != nil {
		t.Error("could not create document")
	}

	if utf8.RuneCountInString(id) == 0 {
		t.Error("unable to return id")
	}

}

// TestDeleteDocument creates a new document and then deletes it after a delay
func TestDeleteDocument(t *testing.T) {
	fs := firebase.FirestoreContext{}
	err := fs.Initialize(serviceAccountPath)
	defer fs.Close()
	if err != nil {
		t.Error("could not initialize")
	}

	testData := map[string]interface{}{
		"aaa": 1,
	}

	newDoc, _ := fs.AddDocument(testCollection, testData)
	// TODO remove pause?
	// small pause to see the doc appear in firebase
	time.Sleep(3 * time.Second)

	err = fs.DeleteDocument(testCollection, newDoc)
	if err != nil {
		t.Error("could not delete document")
	}
}

// TestDeleteNonExistingDocument tries to delete a document that does not exist
func TestDeleteNonExistingDocument(t *testing.T) {
	fs := firebase.FirestoreContext{}
	err := fs.Initialize(serviceAccountPath)
	defer fs.Close()
	if err != nil {
		t.Error("could not initialize")
	}

	err = fs.DeleteDocument(testCollection, "non-existingId")
	if err != nil {
		t.Error("could not delete document")
	}
}

// TestReadDocument reads document with known content
func TestReadDocument(t *testing.T) {
	fs := firebase.FirestoreContext{}
	err := fs.Initialize(serviceAccountPath)
	defer fs.Close()
	if err != nil {
		t.Error("could not initialize")
	}

	testData := map[string]interface{}{
		"a":   1,
		"bb":  "two",
		"ccc": 3.0,
	}

	newDoc, err := fs.AddDocument(testCollection, testData)
	if err != nil {
		t.Error("unable to create new document")
	}

	content, err := fs.ReadDocument(testCollection, newDoc)
	if err != nil {
		t.Error("unable to read document")
	}
	// TODO safer equality testing?
	out := fmt.Sprint(testData)
	in := fmt.Sprint(content)
	if out != in {
		t.Error("could not get back equal data")
	}
}

// TestCountDocuments
func TestCountDocuments(t *testing.T) {
	fs := firebase.FirestoreContext{}
	err := fs.Initialize(serviceAccountPath)
	defer fs.Close()
	if err != nil {
		t.Error("could not initialize")
	}
	// Create new collection with 5 docs:
	newCollection := "test123"
	var ids []string
	for i := 0; i < 5; i++ {
		id, _ := fs.AddDocument(newCollection, map[string]interface{}{"a": i + 1})
		ids = append(ids, id)
	}

	count, err := fs.CountDocuments(newCollection)
	if err != nil {
		t.Error("unable to count")
	}
	fmt.Println("Number of docs: ", count)
	if count != 5 {
		t.Error("wrong number of docs")
	}
	// TODO remove pause
	time.Sleep(10 * time.Second)
	// delete newly created docs:
	for _, id := range ids {
		err = fs.DeleteDocument(newCollection, id)
		if err != nil {
			t.Error("unable to delete")
		}
	}

	count, err = fs.CountDocuments(newCollection)
	if err != nil {
		t.Error("unable to count")
	}
	if count != 0 {
		t.Error("wrong number of docs")
	}

}
