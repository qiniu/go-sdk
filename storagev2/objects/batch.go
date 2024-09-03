package objects

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gammazero/toposort"
	clientv1 "github.com/qiniu/go-sdk/v7/client"
	internal_context "github.com/qiniu/go-sdk/v7/internal/context"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/batch_ops"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/stat_object"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type (
	// 批处理执行器
	BatchOpsExecutor interface {
		ExecuteBatchOps(context.Context, []Operation, *apis.Storage) error
	}

	// 串行批处理执行器选项
	SerialBatchOpsExecutorOptions struct {
		RetryMax  uint // 最大重试次数，默认为 10
		BatchSize uint // 批次大小，默认为 1000
	}

	serialBatchOpsExecutor struct {
		options *SerialBatchOpsExecutorOptions
	}

	// 并行批处理执行器选项
	ConcurrentBatchOpsExecutorOptions struct {
		RetryMax          uint          // 最大重试次数，默认为 10
		InitBatchSize     uint          // 初始批次大小，默认为 250
		MaxBatchSize      uint          // 最大批次大小，默认为 250
		MinBatchSize      uint          // 最小批次大小，默认为 50
		DoublingFactor    uint          // 批次大小翻倍系数，默认为 2
		DoublingInterval  time.Duration // 翻倍时间间隔，默认为 1 分钟
		InitWorkers       uint          // 初始化并发数，默认为 20
		MaxWorkers        uint          // 最大并发数，默认为 20
		MinWorkers        uint          // 最小并发数，默认为 1
		AddWorkerInterval time.Duration // 增加并发数时间间隔，默认为 1 分钟
	}

	concurrentBatchOpsExecutor struct {
		options *ConcurrentBatchOpsExecutorOptions
	}

	operation struct {
		Operation
		tries uint
	}

	requestsManager struct {
		storage                                                         *apis.Storage
		lock                                                            sync.Mutex
		operations                                                      [][]*operation
		batchSize, minBatchSize, maxBatchSize, doublingFactor, maxTries uint
		doublingInterval                                                time.Duration
		ticker                                                          *time.Ticker
		resetTicker                                                     chan struct{}
		cancelTicker                                                    internal_context.CancelCauseFunc
		lastDecreaseBatchSizeTime                                       time.Time
		lastDecreaseBatchSizeTimeMutex                                  sync.Mutex
		waitGroup                                                       sync.WaitGroup
	}

	workersManager struct {
		lock                                  sync.Mutex
		parentCtx                             internal_context.Context
		parentCancelFunc                      internal_context.CancelCauseFunc
		cancels                               []internal_context.CancelCauseFunc
		requestsManager                       *requestsManager
		maxWorkers, minWorkers                uint
		addWorkerInterval                     time.Duration
		ticker                                *time.Ticker
		resetTicker                           chan struct{}
		cancelTickerFunc                      internal_context.CancelCauseFunc
		lastResetTickerTime                   time.Time
		lastResetTickerTimeMutex              sync.Mutex
		timerWaitGroup, asyncWorkersWaitGroup sync.WaitGroup
	}
)

// 创建串型批处理执行器
func NewSerialBatchOpsExecutor(options *SerialBatchOpsExecutorOptions) BatchOpsExecutor {
	if options == nil {
		options = &SerialBatchOpsExecutorOptions{}
	}
	return &serialBatchOpsExecutor{options}
}

func (executor *serialBatchOpsExecutor) ExecuteBatchOps(ctx context.Context, operations []Operation, storage *apis.Storage) error {
	ops := make([]*operation, len(operations))
	for i, op := range operations {
		ops[i] = &operation{Operation: op}
	}
	_, err := doOperations(ctx, ops, storage, executor.options.BatchSize, executor.options.RetryMax)
	return err
}

// 创建并行批处理执行器
func NewConcurrentBatchOpsExecutor(options *ConcurrentBatchOpsExecutorOptions) BatchOpsExecutor {
	if options == nil {
		options = &ConcurrentBatchOpsExecutorOptions{}
	}
	return &concurrentBatchOpsExecutor{options}
}

func (executor *concurrentBatchOpsExecutor) ExecuteBatchOps(ctx context.Context, operations []Operation, storage *apis.Storage) error {
	rm, err := newRequestsManager(
		storage,
		executor.options.InitBatchSize,
		executor.options.MinBatchSize,
		executor.options.MaxBatchSize,
		executor.options.DoublingFactor,
		executor.options.RetryMax,
		executor.options.DoublingInterval,
		operations,
	)
	if err != nil {
		return err
	}
	defer rm.done()
	wm := newWorkersManager(
		ctx,
		executor.options.InitWorkers,
		executor.options.MinWorkers,
		executor.options.MaxWorkers,
		executor.options.AddWorkerInterval,
		rm,
	)
	return wm.wait()
}

func newRequestsManager(storage *apis.Storage, initBatchSize, minBatchSize, maxBatchSize, doublingFactor, maxTries uint, doublingInterval time.Duration, operations []Operation) (*requestsManager, error) {
	if initBatchSize == 0 {
		initBatchSize = 250
	}
	if minBatchSize == 0 {
		minBatchSize = 50
	}
	if maxBatchSize == 0 {
		maxBatchSize = 250
	}
	if maxBatchSize < minBatchSize {
		maxBatchSize = minBatchSize
	}
	if initBatchSize < minBatchSize {
		initBatchSize = minBatchSize
	}
	if initBatchSize > maxBatchSize {
		initBatchSize = maxBatchSize
	}
	if doublingFactor < 2 {
		doublingFactor = 2
	}
	if doublingInterval == 0 {
		doublingInterval = 1 * time.Minute
	}
	if maxTries == 0 {
		maxTries = 10
	}

	sortedOperations, err := topoSort(operations)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := internal_context.WithCancelCause(internal_context.Background())
	rm := requestsManager{
		storage:          storage,
		operations:       wrapOperations(filterOperations(sortedOperations)),
		batchSize:        initBatchSize,
		minBatchSize:     minBatchSize,
		maxBatchSize:     maxBatchSize,
		doublingFactor:   doublingFactor,
		doublingInterval: doublingInterval,
		ticker:           time.NewTicker(doublingInterval),
		resetTicker:      make(chan struct{}, 1024),
		cancelTicker:     cancelFunc,
	}
	sortOperations(rm.operations)

	rm.waitGroup.Add(1)
	go rm.asyncLoop(ctx)
	return &rm, nil
}

func (rm *requestsManager) asyncLoop(ctx internal_context.Context) {
	defer rm.waitGroup.Done()

	for {
		select {
		case <-rm.resetTicker:
			// do nothing
		case <-rm.ticker.C:
			rm.increaseBatchSize()
		case <-ctx.Done():
			return
		}
	}
}

func (rm *requestsManager) done() {
	rm.cancelTicker(nil)
	rm.ticker.Stop()
	rm.waitGroup.Wait()
}

func (rm *requestsManager) takeOperations() []*operation {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	needed := int(rm.batchSize)
	got := make([]*operation, 0, needed)
	for needed > 0 && len(rm.operations) > 0 {
		foundOperations, restOperations := findBestMatches(needed, rm.operations)
		if len(foundOperations) == 0 {
			break
		}
		rm.operations = restOperations
		got = append(got, foundOperations...)
		needed -= len(foundOperations)
	}
	if len(got) == 0 && needed > 0 && len(rm.operations) > 0 { // 一个都没获得，但依然有剩余的操作还没能获取，说明锁有剩余的操作组的大小都大于 batchSize
		got = rm.operations[0]
		rm.operations = rm.operations[1:]
	}
	return got
}

func (rm *requestsManager) putBackOperations(operations []*operation) {
	if len(operations) == 0 {
		return
	}

	rm.lock.Lock()
	defer rm.lock.Unlock()

	rm.operations = append(rm.operations, operations)
	sortOperations(rm.operations)
}

func (rm *requestsManager) isOperationsEmpty() bool {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	return len(rm.operations) == 0
}

func (rm *requestsManager) handleTimeoutError() {
	rm.lastDecreaseBatchSizeTimeMutex.Lock()
	defer rm.lastDecreaseBatchSizeTimeMutex.Unlock()

	canDecrease := time.Since(rm.lastDecreaseBatchSizeTime) > time.Second
	if canDecrease {
		rm.decreaseBatchSize()
		rm.lastDecreaseBatchSizeTime = time.Now()
	}
}

func (rm *requestsManager) decreaseBatchSize() {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	batchSize := rm.batchSize / rm.doublingFactor
	if batchSize < rm.minBatchSize {
		batchSize = rm.minBatchSize
	}
	rm.batchSize = batchSize
	rm.ticker.Stop()
	rm.ticker = time.NewTicker(rm.doublingInterval)
	rm.resetTicker <- struct{}{}
}

func (rm *requestsManager) increaseBatchSize() {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	batchSize := rm.batchSize * rm.doublingFactor
	if batchSize > rm.maxBatchSize {
		batchSize = rm.maxBatchSize
	}
	rm.batchSize = batchSize
}

func newWorkersManager(ctx internal_context.Context, initWorkers, minWorkers, maxWorkers uint, addWorkerInterval time.Duration, requestsManager *requestsManager) *workersManager {
	if initWorkers == 0 {
		initWorkers = 20
	}
	if minWorkers == 0 {
		minWorkers = 1
	}
	if maxWorkers == 0 {
		maxWorkers = 20
	}
	if maxWorkers < minWorkers {
		maxWorkers = minWorkers
	}
	if initWorkers < minWorkers {
		initWorkers = minWorkers
	}
	if initWorkers > maxWorkers {
		initWorkers = maxWorkers
	}
	if addWorkerInterval == 0 {
		addWorkerInterval = 1 * time.Minute
	}
	wm := new(workersManager)
	wm.parentCtx, wm.parentCancelFunc = internal_context.WithCancelCause(ctx)
	wm.requestsManager = requestsManager
	wm.cancels = make([]internal_context.CancelCauseFunc, initWorkers)
	wm.maxWorkers = maxWorkers
	wm.minWorkers = minWorkers
	wm.addWorkerInterval = addWorkerInterval
	wm.ticker = time.NewTicker(addWorkerInterval)
	wm.resetTicker = make(chan struct{}, 1024)

	var timerTickerCtx internal_context.Context
	timerTickerCtx, wm.cancelTickerFunc = internal_context.WithCancelCause(wm.parentCtx)

	wm.timerWaitGroup.Add(1)
	go wm.asyncAddWorkersLoop(timerTickerCtx)

	for i := uint(0); i < initWorkers; i++ {
		workerCtx, workerCancelFunc := internal_context.WithCancelCause(wm.parentCtx)
		wm.cancels[i] = workerCancelFunc
		wm.asyncWorkersWaitGroup.Add(1)
		go wm.asyncWorker(workerCtx, i)
	}
	return wm
}

func (wm *workersManager) asyncAddWorkersLoop(ctx internal_context.Context) {
	defer wm.timerWaitGroup.Done()

	for {
		select {
		case <-wm.resetTicker:
			// do nothing
		case _, ok := <-wm.ticker.C:
			if !ok {
				return
			}
			if wm.getWorkersCount() < wm.maxWorkers {
				wm.spawnWorker()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (wm *workersManager) wait() error {
	wm.asyncWorkersWaitGroup.Wait()
	wm.ticker.Stop()
	wm.cancelTickerFunc(nil)
	wm.timerWaitGroup.Wait()
	if wm.requestsManager.isOperationsEmpty() {
		return wm.parentCtx.Err()
	}
	return wm.doOperationsSync()
}

func (wm *workersManager) doOperationsSync() error {
	for {
		if err := getCtxError(wm.parentCtx); err != nil {
			return err
		}
		if operations := wm.requestsManager.takeOperations(); len(operations) > 0 {
			if operations, err := wm.doOperations(wm.parentCtx, operations); err != nil {
				wm.requestsManager.putBackOperations(operations)
				if isTimeoutError(err) {
					wm.requestsManager.handleTimeoutError()
				} else {
					wm.setError(err)
					return err
				}
			}
		} else {
			return nil
		}
	}
}

func (wm *workersManager) asyncWorker(ctx internal_context.Context, id uint) {
	defer wm.asyncWorkersWaitGroup.Done()
	for getCtxError(wm.parentCtx) == nil {
		if operations := wm.requestsManager.takeOperations(); len(operations) > 0 {
			if operations, err := wm.doOperations(ctx, operations); err != nil {
				// 确定错误是否可以重试
				wm.requestsManager.putBackOperations(operations)
				if isTimeoutError(err) { // 超时，说明 batchSize 过大
					wm.requestsManager.handleTimeoutError()
				} else if isOutOfQuotaError(err) {
					// 并发度过高，自杀减少并发度
					if wm.handleOutOfQuotaError(id, err) {
						return
					}
				} else {
					wm.setError(err)
					return
				}
			}
		} else {
			break
		}
	}
}

func (wm *workersManager) handleOutOfQuotaError(id uint, err error) bool {
	wm.lastResetTickerTimeMutex.Lock()
	defer wm.lastResetTickerTimeMutex.Unlock()

	canReset := time.Since(wm.lastResetTickerTime) > time.Second
	if canReset { // 这里禁止并行杀死 worker，防止杀死速度过快
		wm.ticker.Stop()
		wm.ticker = time.NewTicker(wm.addWorkerInterval)
		wm.resetTicker <- struct{}{}
		wm.killWorker(id, err)
		wm.lastResetTickerTime = time.Now()
	}
	return canReset
}

func (wm *workersManager) doOperations(ctx internal_context.Context, operations []*operation) ([]*operation, error) {
	return doOperations(ctx, operations, wm.requestsManager.storage, wm.requestsManager.batchSize, wm.requestsManager.maxTries)
}

func (wm *workersManager) setError(err error) {
	wm.parentCancelFunc(err)
}

func (wm *workersManager) getWorkersCount() (count uint) {
	wm.lock.Lock()
	defer wm.lock.Unlock()

	for _, c := range wm.cancels {
		if c != nil {
			count += 1
		}
	}
	return
}

func (wm *workersManager) killWorker(id uint, err error) {
	wm.lock.Lock()
	defer wm.lock.Unlock()

	cancelFunc := wm.cancels[id]
	cancelFunc(err)
	wm.cancels[id] = nil
}

func (wm *workersManager) spawnWorker() {
	wm.lock.Lock()
	defer wm.lock.Unlock()

	workerCtx, workerCancelFunc := internal_context.WithCancelCause(wm.parentCtx)
	for id := range wm.cancels {
		if wm.cancels[id] == nil {
			wm.cancels[id] = workerCancelFunc
			go wm.asyncWorker(workerCtx, uint(id))
			return
		}
	}
	wm.cancels = append(wm.cancels, workerCancelFunc)
	go wm.asyncWorker(workerCtx, uint(len(wm.cancels)-1))
}

func doOperations(ctx internal_context.Context, operations []*operation, storage *apis.Storage, batchSize, maxTries uint) ([]*operation, error) {
	if batchSize == 0 {
		batchSize = 1000
	}
	if maxTries == 0 {
		maxTries = 10
	}
	for len(operations) > 0 {
		thisBatchSize := batchSize
		if thisBatchSize > uint(len(operations)) {
			thisBatchSize = uint(len(operations))
		}
		toDoThisLoop := operations[:thisBatchSize]
		willDoNextLoop := make([]*operation, 0, thisBatchSize)
		bucketName := toDoThisLoop[0].relatedEntries()[0].bucketName

		operationsStrings := make([]string, len(toDoThisLoop))
		for i, operation := range toDoThisLoop {
			operationsStrings[i] = operation.String()
		}

		response, err := storage.BatchOps(ctx, &apis.BatchOpsRequest{
			Operations: operationsStrings,
		}, &apis.Options{
			OverwrittenBucketName: bucketName,
		})
		if err != nil {
			return operations, err
		}
		for i, operationResponse := range response.OperationResponses {
			operation := toDoThisLoop[i]
			if operationResponse.Code == 200 {
				var object ObjectDetails
				if err = object.fromOperationResponseData(operation.relatedEntries()[0].objectName, &operationResponse.Data); err != nil {
					operation.handleResponse(nil, err)
					continue
				}
				operation.handleResponse(&object, nil)
			} else {
				operation.handleResponse(nil, errors.New(operationResponse.Data.Error))
				operation.tries += 1
				if retrier.IsStatusCodeRetryable(int(operationResponse.Code)) && operation.tries < maxTries {
					willDoNextLoop = append(willDoNextLoop, operation)
				}
			}
		}
		if thisBatchSize >= batchSize {
			willDoNextLoop = append(willDoNextLoop, operations[thisBatchSize:]...)
		}
		operations = willDoNextLoop
	}
	return nil, nil
}

func (object *ObjectDetails) fromOperationResponseData(key string, data *batch_ops.OperationResponseData) error {
	var (
		md5 []byte
		err error
	)
	object.Name = key
	object.UploadedAt = time.Unix(data.PutTime/1e7, (data.PutTime%1e7)*1e2)
	object.ETag = data.Hash
	object.Size = data.Size
	object.MimeType = data.MimeType
	object.StorageClass = StorageClass(data.Type)
	object.EndUser = data.EndUser
	object.Status = Status(data.Status)
	object.RestoreStatus = RestoreStatus(data.RestoringStatus)
	object.Metadata = data.Metadata
	if data.Md5 != "" {
		md5, err = hex.DecodeString(data.Md5)
		if err != nil {
			return err
		}
	}
	if len(md5) > 0 {
		copy(object.MD5[:], md5)
	}
	if len(data.Parts) > 0 {
		object.Parts = append(make(stat_object.PartSizes, 0, len(data.Parts)), data.Parts...)
	}
	if data.TransitionToIaTime > 0 {
		transitionToIA := time.Unix(data.TransitionToIaTime, 0)
		object.TransitionToIA = &transitionToIA
	}
	if data.TransitionToArchiveIrTime > 0 {
		transitionToArchiveIR := time.Unix(data.TransitionToArchiveIrTime, 0)
		object.TransitionToArchiveIR = &transitionToArchiveIR
	}
	if data.TransitionToArchiveTime > 0 {
		transitionToArchive := time.Unix(data.TransitionToArchiveTime, 0)
		object.TransitionToArchive = &transitionToArchive
	}
	if data.TransitionToDeepArchiveTime > 0 {
		transitionToDeepArchive := time.Unix(data.TransitionToDeepArchiveTime, 0)
		object.TransitionToDeepArchive = &transitionToDeepArchive
	}
	if data.ExpirationTime > 0 {
		expireAt := time.Unix(data.ExpirationTime, 0)
		object.ExpireAt = &expireAt
	}
	return nil
}

type groups struct {
	keyGroups         []map[string]struct{}
	keyIndexes        map[string]int
	emptyGroupIndexes []int
}

func findKeyFromGroups(g *groups, key string) (int, bool) {
	idx, ok := g.keyIndexes[key]
	return idx, ok
}

func addKeyToGroup(g *groups, key string) {
	if _, ok := findKeyFromGroups(g, key); ok {
		return
	}
	appendKeyToGroup(g, key)
}

func appendKeyToGroup(g *groups, keys ...string) {
	foundIndex := -1
	if len(g.emptyGroupIndexes) > 0 {
		lastIdx := len(g.emptyGroupIndexes) - 1
		foundIndex = g.emptyGroupIndexes[lastIdx]
		g.emptyGroupIndexes = g.emptyGroupIndexes[:lastIdx]
	}
	newKeyGroup := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		newKeyGroup[key] = struct{}{}
	}

	if foundIndex < 0 {
		foundIndex = len(g.keyGroups)
		g.keyGroups = append(g.keyGroups, newKeyGroup)
	} else {
		g.keyGroups[foundIndex] = newKeyGroup
	}
	for _, key := range keys {
		g.keyIndexes[key] = foundIndex
	}
}

func connectGroup(g *groups, key1, key2 string) {
	k1, ok := findKeyFromGroups(g, key1)
	if !ok {
		k1 = -1
	}
	k2, ok := findKeyFromGroups(g, key2)
	if !ok {
		k2 = -1
	}
	if k1 == k2 {
		if k1 < 0 {
			appendKeyToGroup(g, key1, key2)
		}
	} else if k1 < 0 {
		g.keyGroups[k2][key1] = struct{}{}
		g.keyIndexes[key1] = k2
	} else if k2 < 0 {
		g.keyGroups[k1][key2] = struct{}{}
		g.keyIndexes[key2] = k1
	} else {
		for k := range g.keyGroups[k2] {
			g.keyGroups[k1][k] = struct{}{}
			g.keyIndexes[k] = k1
		}
		g.keyGroups[k2] = nil
		g.emptyGroupIndexes = append(g.emptyGroupIndexes, k2)
	}
}

func topoSort(operations []Operation) ([][]Operation, error) {
	var (
		edges        = make([]toposort.Edge, 0, len(operations))
		rootNodesMap = make(map[string]int, len(operations)*2)
		g            = groups{keyIndexes: make(map[string]int)}
	)

	for operationId, operation := range operations {
		if operation == nil {
			continue
		}
		edges = append(edges, toposort.Edge{nil, operationId})

		var firstKey string
		for _, relatedEntry := range operation.relatedEntries() {
			key := relatedEntry.String()
			if oldRootOperationId, ok := rootNodesMap[key]; ok {
				edges = append(edges, toposort.Edge{oldRootOperationId, operationId})
			}
			rootNodesMap[key] = operationId

			addKeyToGroup(&g, key)
			if firstKey == "" {
				firstKey = key
			} else {
				connectGroup(&g, firstKey, key)
			}
		}
	}

	sortedOperationIds, err := toposort.Toposort(edges)
	if err != nil {
		return nil, err
	}

	groupedOperations := make([][]Operation, len(g.keyGroups))
	for _, sortedOperationId := range sortedOperationIds {
		operation := operations[sortedOperationId.(int)]
		operationKey := operation.relatedEntries()[0].String()
		index, ok := findKeyFromGroups(&g, operationKey)
		if !ok {
			panic(fmt.Sprintf("failed to find key `%s`, which is unexpected", operationKey))
		}
		groupedOperations[index] = append(groupedOperations[index], operation)
	}
	return groupedOperations, nil
}

func sortOperations(operations [][]*operation) {
	sort.Slice(operations, func(i, j int) bool {
		return len(operations[i]) < len(operations[j])
	})
}

func filterOperations(operationsGroups [][]Operation) [][]Operation {
	results := make([][]Operation, 0, len(operationsGroups))
	for _, operationsGroup := range operationsGroups {
		if len(operationsGroup) == 0 {
			continue
		}
		results = append(results, operationsGroup)
	}
	return results
}

func wrapOperations(operationsGroups [][]Operation) [][]*operation {
	results := make([][]*operation, len(operationsGroups))
	for groupId, operationsGroup := range operationsGroups {
		groupResults := make([]*operation, len(operationsGroup))
		for opId, op := range operationsGroup {
			groupResults[opId] = &operation{Operation: op}
		}
		results[groupId] = groupResults
	}
	return results
}

func findBestMatches(size int, operations [][]*operation) ([]*operation, [][]*operation) {
	var lastIdx int = -1
	sort.Search(len(operations), func(idx int) bool {
		if len(operations[idx]) <= size {
			lastIdx = idx
		}
		return size <= len(operations[idx])
	})
	if lastIdx < 0 {
		return nil, operations
	}
	bestMatches := operations[lastIdx]
	if len(bestMatches) > size {
		return nil, operations
	}
	return bestMatches, append(operations[:lastIdx], operations[lastIdx+1:]...)
}

func getCtxError(ctx internal_context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func isTimeoutError(err error) bool {
	if err == context.DeadlineExceeded {
		return false
	} else if os.IsTimeout(err) {
		return true
	} else if clientErr, ok := unwrapUnderlyingError(err).(*clientv1.ErrorInfo); ok {
		if clientErr.Code == 504 {
			return true
		}
	}
	return false
}

func isOutOfQuotaError(err error) bool {
	if clientErr, ok := unwrapUnderlyingError(err).(*clientv1.ErrorInfo); ok {
		if clientErr.Code == 573 {
			return true
		}
	}
	return false
}

func tryToUnwrapUnderlyingError(err error) (error, bool) {
	switch err := err.(type) {
	case *os.PathError:
		return err.Err, true
	case *os.LinkError:
		return err.Err, true
	case *os.SyscallError:
		return err.Err, true
	case *url.Error:
		return err.Err, true
	case *net.OpError:
		return err.Err, true
	}
	return err, false
}

func unwrapUnderlyingError(err error) error {
	ok := true
	for ok {
		err, ok = tryToUnwrapUnderlyingError(err)
	}
	return err
}
