package cloudygcp

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/appliedres/cloudy"
	cloudystorage "github.com/appliedres/cloudy/storage"
	"google.golang.org/api/iterator"
)

const GoogleCloudStorageDriver = "gcp-storage"

var GoogleCloudStorageLabelRegExValue = regexp.MustCompile("^[a-z-_]*$")

func init() {
	cloudystorage.ObjectStorageProviders.Register(GcpSecretsManager, &GoogleCloudStorageFactory{})
}

type GoogleCloudStorage struct {
	Project string
	Client  *storage.Client
}

type GoogleCloudStorageConfig struct {
	Project string
}

type GoogleCloudStorageFactory struct{}

func (c *GoogleCloudStorageFactory) Create(cfg interface{}) (cloudystorage.ObjectStorageManager, error) {
	sec := cfg.(*GoogleCloudStorageConfig)
	if sec == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}
	return NewGoogleCloudStorage(context.Background(), sec.Project)
}

func (c *GoogleCloudStorageFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
	cfg := &GoogleCloudStorageConfig{}
	cfg.Project = env.Force("GCP_PROJECT")
	return cfg, nil
}

type GoogleCloudStorageBucket struct {
	Project string
	Bucket  string
	Client  *storage.Client
}

func NewGoogleCloudStorage(ctx context.Context, project string) (*GoogleCloudStorage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}

	return &GoogleCloudStorage{
		Project: project,
		Client:  client,
	}, nil
}

// Implements ObjectStorageManager, Each one of these is all the buckets for a project
func (gcps *GoogleCloudStorage) Exists(ctx context.Context, key string) (bool, error) {
	bucket := gcps.Client.Bucket(key)
	_, err := bucket.Attrs(ctx)

	if err != nil {
		if gcps.isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Not sure what to do here. I guess just return true
	return true, nil
}

func (gcps *GoogleCloudStorage) List(ctx context.Context) ([]*cloudystorage.StorageArea, error) {
	var rtn []*cloudystorage.StorageArea
	it := gcps.Client.Buckets(ctx, gcps.Project)
	for {
		battrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		rtn = append(rtn, &cloudystorage.StorageArea{
			Name: battrs.Name,
			Tags: battrs.Labels,
		})

	}
	return rtn, nil
}

func (gcps *GoogleCloudStorage) GetItem(ctx context.Context, key string) (*cloudystorage.StorageArea, error) {
	bucket := gcps.Client.Bucket(key)
	battrs, err := bucket.Attrs(ctx)

	if err != nil {
		return nil, err
	}

	if battrs == nil {
		return nil, nil
	}

	// Not sure what to do here. I guess just return true
	return &cloudystorage.StorageArea{
		Name: battrs.Name,
		Tags: battrs.Labels,
	}, nil
}

func (gcps *GoogleCloudStorage) Get(ctx context.Context, key string) (cloudystorage.ObjectStorage, error) {
	exists, err := gcps.Exists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	return &GoogleCloudStorageBucket{
		Project: gcps.Project,
		Bucket:  key,
		Client:  gcps.Client,
	}, nil
}

func (gcps *GoogleCloudStorage) Create(ctx context.Context, key string, openToPublic bool, tags map[string]string) (cloudystorage.ObjectStorage, error) {

	bucket := gcps.Client.Bucket(key)

	attrs := &storage.BucketAttrs{
		Labels: prepareTags(ctx, tags),
	}

	err := bucket.Create(ctx, gcps.Project, attrs)

	if err != nil {
		return nil, cloudy.Error(ctx, "Bucket(%q).Create: %v", key, err)
	}

	return &GoogleCloudStorageBucket{
		Project: gcps.Project,
		Bucket:  key,
		Client:  gcps.Client,
	}, nil
}

func (gcps *GoogleCloudStorage) Delete(ctx context.Context, key string) error {
	bucket := gcps.Client.Bucket(key)
	err := bucket.Delete(ctx)

	return err
}

func (gcps *GoogleCloudStorage) isNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "bucket doesn't exist")
}

// Implements ObjectStorage. Each instance represents a "bucket" in Google
func (gcpb *GoogleCloudStorageBucket) Upload(ctx context.Context, key string, data io.Reader, tags map[string]string) error {
	//TODO: Save tags

	o := gcpb.Client.Bucket(gcpb.Bucket).Object(key)

	// Optional: set a generation-match precondition to avoid potential race
	// conditions and data corruptions. The request to upload is aborted if the
	// object's generation number does not match your precondition.
	// For an object that does not yet exist, set the DoesNotExist precondition.
	// o = o.If(storage.Conditions{DoesNotExist: true})

	// If the live object already exists in your bucket, set instead a
	// generation-match precondition using the live object's generation number.
	// attrs, err := o.Attrs(ctx)
	// if err != nil {
	//      return fmt.Errorf("object.Attrs: %v", err)
	// }
	// o = o.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	_, err := io.Copy(wc, data)
	if err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	err = wc.Close()
	if err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	_, err = o.Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: prepareTags(ctx, tags),
	})

	return err
}
func (gcpb *GoogleCloudStorageBucket) Exists(ctx context.Context, key string) (bool, error) {
	o := gcpb.Client.Bucket(gcpb.Bucket).Object(key)

	attrs, err := o.Attrs(ctx)
	if err != nil {
		if gcpb.isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return attrs != nil, nil
}
func (gcpb *GoogleCloudStorageBucket) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	o := gcpb.Client.Bucket(gcpb.Bucket).Object(key)
	reader, err := o.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %v", key, err)
	}
	return reader, nil
}

func (gcpb *GoogleCloudStorageBucket) Delete(ctx context.Context, key string) error {
	o := gcpb.Client.Bucket(gcpb.Bucket).Object(key)
	return o.Delete(ctx)
}

func (gcpb *GoogleCloudStorageBucket) List(ctx context.Context, prefix string) ([]*cloudystorage.StoredObject, []*cloudystorage.StoredPrefix, error) {
	var objects []*cloudystorage.StoredObject
	var folders []*cloudystorage.StoredPrefix
	it := gcpb.Client.Bucket(gcpb.Bucket).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: "/",
	})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("Bucket(%q).Objects(): %v", gcpb.Bucket, err)
		}
		if attrs.Name == "" {
			folders = append(folders, &cloudystorage.StoredPrefix{
				Key: attrs.Prefix,
			})
		} else {
			objects = append(objects, &cloudystorage.StoredObject{
				Key:  attrs.Name,
				Tags: attrs.Metadata,
				Size: attrs.Size,
				MD5:  string(attrs.MD5),
			})
		}
	}

	return objects, folders, nil
}

func (gcpb *GoogleCloudStorageBucket) isNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "Error 403:")
}

func prepareTags(ctx context.Context, tags map[string]string) map[string]string {
	m := make(map[string]string)

	for k, v := range tags {
		k1 := strings.ToLower(k)
		v1 := strings.ToLower(v)

		if GoogleCloudStorageLabelRegExValue.MatchString(k1) && GoogleCloudStorageLabelRegExValue.MatchString(v1) {
			m[k1] = v1
		} else {
			cloudy.Error(ctx, "Invalid Label found, skipping, %v | %v", k1, v1)
		}
	}
	return m
}
