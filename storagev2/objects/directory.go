package objects

import (
	"container/list"
	"path/filepath"
	"strings"

	"github.com/qiniu/go-sdk/v7/internal/context"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
)

type (
	Directory struct {
		bucket            *Bucket
		prefix, delimiter string
	}

	DirectoryEntry struct {
		DirectoryName string
		Object        *ObjectDetails
	}

	ListEntriesOptions struct {
		NeedParts bool
		Recursive bool
	}
)

var SkipDir = filepath.SkipDir

func (directory *Directory) MoveTo(ctx context.Context, toBucketName, toPrefix string) error {
	if !strings.HasSuffix(toPrefix, directory.delimiter) {
		toPrefix += directory.delimiter
	}
	operations := make([]Operation, 0, 16)
	if err := directory.forEachObject(ctx, func(objectDetails *ObjectDetails) {
		toObjectName := toPrefix + strings.TrimPrefix(objectDetails.Name, directory.prefix)
		operations = append(operations, directory.bucket.Object(objectDetails.Name).MoveTo(toBucketName, toObjectName))
	}); err != nil {
		return err
	}
	return directory.bucket.objectsManager.Batch(ctx, operations, nil)
}

func (directory *Directory) CopyTo(ctx context.Context, toBucketName, toPrefix string) error {
	if !strings.HasSuffix(toPrefix, directory.delimiter) {
		toPrefix += directory.delimiter
	}
	operations := make([]Operation, 0, 16)
	if err := directory.forEachObject(ctx, func(objectDetails *ObjectDetails) {
		toObjectName := toPrefix + strings.TrimPrefix(objectDetails.Name, directory.prefix)
		operations = append(operations, directory.bucket.Object(objectDetails.Name).CopyTo(toBucketName, toObjectName))
	}); err != nil {
		return err
	}
	return directory.bucket.objectsManager.Batch(ctx, operations, nil)
}

func (directory *Directory) Delete(ctx context.Context) error {
	operations := make([]Operation, 0, 16)
	if err := directory.forEachObject(ctx, func(objectDetails *ObjectDetails) {
		operations = append(operations, directory.bucket.Object(objectDetails.Name).Delete())
	}); err != nil {
		return err
	}
	return directory.bucket.objectsManager.Batch(ctx, operations, nil)
}

func (directory *Directory) forEachObject(ctx context.Context, each func(*ObjectDetails)) error {
	lister := directory.bucket.List(ctx, &ListObjectsOptions{Prefix: directory.prefix})
	defer lister.Close()

	var objectDetails ObjectDetails
	for lister.Next(&objectDetails) {
		each(&objectDetails)
	}

	return lister.Error()
}

func (directory *Directory) ListEntries(ctx context.Context, options *ListEntriesOptions, f func(*DirectoryEntry) error) error {
	if options == nil {
		options = &ListEntriesOptions{}
	}

	directories := list.New()
	directories.PushBack(directory)
	return consumeEntries(ctx, directories, options, f)
}

func (directory *Directory) listEntries(ctx context.Context, marker string, needParts bool) ([]*DirectoryEntry, string, error) {
	entries := make([]*DirectoryEntry, 0, 1024)
	request := apis.GetObjectsRequest{
		Bucket:    directory.bucket.name,
		Prefix:    directory.prefix,
		Delimiter: directory.delimiter,
		Marker:    marker,
		NeedParts: needParts,
	}
	response, err := directory.bucket.objectsManager.storage.GetObjects(ctx, &request, nil)
	if err != nil {
		return nil, "", err
	}
	for _, commonPrefix := range response.CommonPrefixes {
		entries = append(entries, &DirectoryEntry{DirectoryName: commonPrefix})
	}
	for _, item := range response.Items {
		objectDetails := new(ObjectDetails)
		if err = objectDetails.fromListedObjectEntry(&item); err != nil {
			return nil, "", err
		}
		entries = append(entries, &DirectoryEntry{Object: objectDetails})
	}
	return entries, response.Marker, nil
}

func consumeEntries(ctx context.Context, directories *list.List, options *ListEntriesOptions, f func(*DirectoryEntry) error) error {
	var firstElement *list.Element
	for {
		if firstElement = directories.Front(); firstElement == nil {
			break
		}
		directories.Remove(firstElement)
		currentDirectory := firstElement.Value.(*Directory)

		var (
			entries   []*DirectoryEntry
			firstPage = true
			marker    string
			err       error
		)
	nextPage:
		for firstPage || marker != "" {
			firstPage = false
			if entries, marker, err = currentDirectory.listEntries(ctx, marker, options.NeedParts); err != nil {
				return err
			} else {
				for _, entry := range entries {
					err := f(entry)
					switch err {
					case nil:
						if options.Recursive && entry.DirectoryName != "" {
							directories.PushBack(&Directory{bucket: currentDirectory.bucket, prefix: entry.DirectoryName, delimiter: currentDirectory.delimiter})
						}
					case SkipDir:
						if entry.DirectoryName == "" {
							continue nextPage
						}
					default:
						return err
					}
				}
			}
		}
	}
	return nil
}
