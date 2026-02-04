# Log Ingestion Service

A distributed log ingestion pipeline built in Go for collecting and processing application logs from multiple directories.

## Overview

This project implements a microservices-based system that discovers, parses, and aggregates `.log` files across distributed file systems. The architecture handles incremental processing — only ingesting new or modified files rather than re-processing everything on each scan.

Key features:

- Concurrent directory scanning with backpressure control
- Incremental file discovery (only process what's changed)
- Stateful ingestion tracking (resume from byte offsets)
- Buffered channel-based communication between services

---

## Pipeline Architecture

Discovery → Processing → Aggregation → Output

### 1. Discovery Layer

Scans configured directories, identifies `.log` files, and emits file metadata (path, size, modification time) to the processing layer via a buffered channel.

### 2. Processing Layer

Reads files from discovery, checks ingestion state, parses log entries, and tracks byte offsets for incremental reads.

### 3. Aggregation Layer

Transforms, enriches, and filters parsed log entries before forwarding to output destinations.

### 4. Output Layer

Stores processed logs in the target system (database, search index, time-series store, etc.).

---

## Completed: Discovery Layer

The directory walker service is complete. It handles the first stage of the pipeline.

### What It Does

- Scans multiple directories concurrently (configurable worker count)
- Filters for `.log` files before statting
- Stats each file to get size and modification time
- Emits `LogFileEntry{Path, Size, ModTime}` to a buffered output channel
- Applies backpressure when the channel is full (workers block instead of flooding memory)

### Design Highlights

**Efficient scanning**: Uses `os.ReadDir` which returns sorted results, enabling binary search to skip already-processed files.

**Filtered statting**: Checks file extension before calling `entry.Info()`, avoiding unnecessary syscalls on non-log files.

**Semaphore-based concurrency**: Caps the number of concurrent directory scans to prevent I/O contention.

**Stateless discovery**: The walker doesn't track ingestion state — that's the processing layer's job. This keeps boundaries clean and prevents silent data loss if the walker crashes.

### Configuration

```json
{
  "walker": {
    "log_dirs": [
      "/dir1",
      "/dir2"
    ],
    "max_discovery_workers": 10    
  },
  "discovered_files_channel_size" : 500
}
```

## Next Steps

### Processing Layer (In Progress)

The ingestion service will:

- Read `LogFileEntry` from the walker's output channel
- Check each file against a persistent state store (path + size + modTime)
- For new files: ingest from byte 0
- For modified files: seek to stored byte offset, ingest only new content
- Parse log lines and forward to aggregation
- Update state store with new byte offset after successful ingestion

### Aggregation Layer (Planned)

- Transform raw log lines into structured events
- Enrich with metadata (service name, environment, timestamps)
- Filter based on severity or pattern matching
- Forward to output layer

### Output Layer (Planned)

- Index logs in Elasticsearch or similar
- Store in time-series database
- Forward to monitoring/alerting systems

---

## License

MIT
