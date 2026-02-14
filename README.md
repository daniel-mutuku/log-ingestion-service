# Log Ingestion Service

A distributed log ingestion pipeline built in Go for collecting and processing application logs from multiple directories.

## Overview

This project implements a microservices-based system that discovers, parses, and aggregates `.log` files across distributed file systems. The service processes files concurrently with proper backpressure control and graceful shutdown handling.

**Current Status:** Core pipeline complete (Discovery → Ingestion → Aggregation). State persistence for incremental processing is the next milestone.

Key features:

- Context-based graceful shutdown (no goroutine leaks)
- Concurrent directory scanning with semaphore limiting
- Stream-based file processing (constant memory footprint)
- Buffered channel communication between pipeline stages

---

## Pipeline Architecture

```
Discovery → Ingestion → Aggregation → Output
```

### 1. Discovery Layer

Scans configured directories, identifies `.log` files, and emits file metadata (path, size, modification time) to the ingestion layer via a buffered channel.

**What it does:**

- Scans multiple directories concurrently (configurable worker count)
- Filters for `.log` files before statting (avoids unnecessary syscalls)
- Stats each file to get size and modification time
- Emits `LogFileEntry{Path, Size, ModTime}` to a buffered output channel
- Applies backpressure when the channel is full
- Respects context cancellation for clean shutdown

**Key design decisions:**

- Uses `os.ReadDir` (returns sorted results for future binary search optimization)
- Semaphore pattern caps concurrent directory scans to prevent I/O contention
- Stateless design — doesn't track ingestion state (separation of concerns)

### 2. Ingestion Layer

Reads discovered files from the channel, streams them line by line, parses log entries, and counts errors by severity level.

**What it does:**

- Multiple workers pull files from the discovery channel concurrently
- Streams files using `bufio.Scanner` (constant memory, no matter the file size)
- Parses each log line and extracts severity level
- Counts ERRORs and WARNs per file
- Sends counts to the aggregation channel
- Respects context cancellation mid-stream (interrupts even during large file processing)

**Key design decisions:**

- Line-by-line streaming prevents memory exhaustion on large files
- Context checks on every iteration enable fast shutdown
- Workers are stateless — no shared memory, just channels

### 3. Aggregation Layer 

Merges error counts from multiple ingestion workers and prints final totals.

**What it does:**

- Receives counts from all ingestion workers via a channel
- Merges counts into running totals
- Prints final statistics when the channel closes or context cancels
- Handles both graceful completion and early shutdown

**Key design decisions:**

- Single aggregator (no lock contention on shared state)
- Waits for channel close to know all workers finished
- Always prints accumulated totals, even on interrupted shutdown

### 4. Output Layer

Store processed logs in a persistent system.

**Planned features:**

- Index logs in Elasticsearch or similar search engine
- Store in time-series database for querying
- Forward to monitoring/alerting systems

---

## Configuration

```json
{
  "walker": {
    "log_dirs": [
      "/var/logs/app",
      "/var/logs/errors"
    ],
    "max_discovery_workers": 10    
  },
  "discovered_files_channel_size": 500,
  "ingestion": {
    "worker_count": 5
  }
}
```

**Config parameters:**

- `log_dirs`: Directories to scan for `.log` files
- `max_discovery_workers`: Max concurrent directory scans (semaphore cap)
- `discovered_files_channel_size`: Buffer size for backpressure control
- `worker_count`: Number of concurrent ingestion workers

---

## Running the Service

```bash
# Build
go build -o log-ingestion

# Run
./log-ingestion

# Graceful shutdown
# Press Ctrl+C — context cancels, all goroutines exit cleanly
```

---

## What's Next

### State Persistence (Next Milestone)

The service currently processes all files from scratch on every run. The next phase adds incremental processing:

**Planned features:**

- Persistent state store (BoltDB or SQLite) tracking processed files
- Store: file path, byte offset, size, modification time
- For new files: start from byte 0
- For modified files: seek to last byte offset, read only new content
- Handle edge cases: file truncation, deletion, rotation

**Why this matters:**

- Process only what changed (1,000 files → 1,200 files = process 200, not 1,200)
- Resume interrupted ingestion exactly where it left off
- Efficient for continuously growing log directories

---

## Blog Series

This project is documented as a learning-in-public series on Medium:

- **Part 1**: [Architecture and Problem Statement]("https://medium.com/@daniel.mutuku404/building-a-log-ingestion-service-in-go-2c10ed836eba")
- **Part 2**: [The Directory Walker (Discovery Layer)]("https://medium.com/@daniel.mutuku404/building-a-log-ingestion-service-in-go-part-2-the-directory-walker-b089e5e58e0a)
- **Part 3**: [Processing and Aggregation (Ingestion + Context)]("https://medium.com/@daniel.mutuku404/building-a-log-ingestion-service-in-go-part-3-processing-and-aggregation-f63c78da186e")
- **Part 4**: State Persistence and Incremental Ingestion *(coming soon)*

---

## Design Philosophy

**Channels over shared memory**: Each pipeline stage communicates only through channels. No mutexes, no shared state.

**Context for coordination**: Every goroutine respects the same context. One cancellation signal stops the entire pipeline cleanly.

**Streaming over batching**: Files are processed line-by-line, not loaded into memory. Handles arbitrarily large files with constant memory footprint.

**Clean boundaries**: Each component has a single responsibility. Discovery discovers. Ingestion processes. Aggregation merges. No overlap.

---

## License

MIT
