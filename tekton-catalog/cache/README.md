# Cache: Reuse the results from previous execution for custom tasks.

### How To

1. Setup.
```go
    taskCacheStore := TaskCacheStore{}
	err := taskCacheStore.Connect(db.ConnectionParams{DbDriver: "sqlite3", DbName: ":memory:"})
	// Currently, mysql and sqlite3 are supported driver.
```

2. Store an entry to cache.
```go
    taskCache := &model.TaskCache{
        TaskHashKey: cacheKey,
        TaskOutput:  cacheOutput,
    }
    taskCacheStore.Put(taskCache)
```

3. Fetch an entry from cache.
```go
     cacheResult, err := taskCacheStore.Get(taskCache.TaskHashKey)
        if err != nil {
            fmt.Printf("%v", err)
        }
     
```
4. Prune entries older than a day using:
```go
    taskCacheStore.PruneOlderThan(time.Now().Add(-24*time.Hour))
```