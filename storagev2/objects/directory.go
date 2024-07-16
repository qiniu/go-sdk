package objects

import (
	"container/list"
	"path/filepath"
	"strings"

	"github.com/qiniu/go-sdk/v7/internal/context"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
)

type (
	// 目录
	Directory struct {
		bucket                *Bucket
		prefix, pathSeparator string
	}

	// 条目
	Entry struct {
		// 目录名称，仅当条目为目录时才有效
		DirectoryName string

		// 对象元信息，仅当条目为对象时才有效
		Object *ObjectDetails
	}

	// 列举条目选项
	ListEntriesOptions struct {
		NeedParts bool // 是否需要分片信息
		Recursive bool // 是否递归列举
	}
)

var SkipDir = filepath.SkipDir

// 移动目录
func (directory *Directory) MoveTo(ctx context.Context, toBucketName, toPrefix string) error {
	if !strings.HasSuffix(toPrefix, directory.pathSeparator) {
		toPrefix += directory.pathSeparator
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

// 复制目录
func (directory *Directory) CopyTo(ctx context.Context, toBucketName, toPrefix string) error {
	if !strings.HasSuffix(toPrefix, directory.pathSeparator) {
		toPrefix += directory.pathSeparator
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

// 删除目录
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

// 列举目录条目
func (directory *Directory) ListEntries(ctx context.Context, options *ListEntriesOptions, f func(*Entry) error) error {
	if options == nil {
		options = &ListEntriesOptions{}
	}

	directories := list.New()
	directories.PushBack(directory)
	return consumeEntries(ctx, directories, options, f)
}

func (directory *Directory) listEntries(ctx context.Context, marker string, needParts bool) ([]*Entry, string, error) {
	entries := make([]*Entry, 0, 1024)
	request := apis.GetObjectsRequest{
		Bucket:    directory.bucket.name,
		Prefix:    directory.prefix,
		Delimiter: directory.pathSeparator,
		Marker:    marker,
		NeedParts: needParts,
	}
	response, err := directory.bucket.objectsManager.storage.GetObjects(ctx, &request, nil)
	if err != nil {
		return nil, "", err
	}
	for _, commonPrefix := range response.CommonPrefixes {
		entries = append(entries, &Entry{DirectoryName: commonPrefix})
	}
	for _, item := range response.Items {
		objectDetails := new(ObjectDetails)
		if err = objectDetails.fromListedObjectEntry(&item); err != nil {
			return nil, "", err
		}
		entries = append(entries, &Entry{Object: objectDetails})
	}
	return entries, response.Marker, nil
}

func consumeEntries(ctx context.Context, directories *list.List, options *ListEntriesOptions, f func(*Entry) error) error {
	var firstElement *list.Element
	for {
		if firstElement = directories.Front(); firstElement == nil {
			break
		}
		directories.Remove(firstElement)
		currentDirectory := firstElement.Value.(*Directory)

		var (
			entries   []*Entry
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
							directories.PushBack(&Directory{bucket: currentDirectory.bucket, prefix: entry.DirectoryName, pathSeparator: currentDirectory.pathSeparator})
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
