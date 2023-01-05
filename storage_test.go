package cloudygcp

import (
	"log"
	"testing"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/testutil"
)

func TestBlobAccount(t *testing.T) {
	ctx := cloudy.StartContext()
	bsa, err := NewGoogleCloudStorage(ctx, "arklouddev")
	if err != nil {
		log.Fatal(err)
		// t.FailNow()
	}

	testutil.TestObjectStorageManager(t, bsa)

	// bucket, err := bsa.Get(ctx, "arkloud-sample")
	// if err != nil {
	// 	log.Fatalf("%v", err)
	// }
	// files, folders, err := bucket.List(ctx, "")
	// if err != nil {
	// 	log.Fatalf("%v", err)
	// }

	// fmt.Printf("FILES----------\n")
	// for _, file := range files {
	// 	fmt.Printf("%v\n", file.Key)
	// }
	// fmt.Printf("\n")
	// fmt.Printf("FOLDERS----------\n")
	// for _, folder := range folders {
	// 	fmt.Printf("%v\n", folder.Key)
	// }
}

// func TestBlobFileAccount(t *testing.T) {
// 	ctx := cloudy.StartContext()
// 	bfa, err := NewGoogleCloudStorage(ctx, "arklouddev")
// 	if err != nil {
// 		log.Fatal(err)
// 		// t.FailNow()
// 	}
// 	testutil.TestFileShareStorageManager(t, bfa, "file-storage-test")

// }
