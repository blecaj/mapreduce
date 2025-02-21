
![MapReduce Framework](https://lyusungwon.github.io/assets/images/mr1.png)

# MapReduce Framework Project

This project implements a simple MapReduce framework in Go, inspired by distributed systems courses like MIT's 6.5840. It serves as both a learning tool and a base for building distributed applications.

## Overview

**What the Framework Does:**

- **Distributed Computation:**  
  Supports a distributed environment by coordinating multiple worker processes that can run on different machines, enabling true scalability and fault tolerance.
  
- **Parallel Processing:**  
  Enables parallel execution of tasks, where multiple workers process map and reduce operations concurrently.
  
- **Fault Tolerance:**  
  Implements a simple mechanism to recover from worker failures by reassigning uncompleted tasks.
  
- **Modularity:**  
  Separates the concerns of task scheduling (handled by the coordinator) and task execution (handled by the workers), making the system easier to maintain and extend.
  
- **Plugin-Based MapReduce Applications:**  
  Supports dynamic loading of application-specific Map and Reduce functions via Go plugins, allowing you to run different applications like word count or text indexing without modifying the core framework.

The project is organized into three main directories:
- **`main/`**: Contains the entry points for running different roles (e.g., coordinator, worker).
- **`mr/`**: Contains the core logic for the MapReduce framework, including task scheduling and job management.
- **`mrapps/`**: Provides sample applications that implement custom Map and Reduce functions (e.g., word count, inverted index).

## Understanding the Architecture

### Coordinator Process
- **Input Management:** Accepts input files (each file is one "split" for a Map task).
- **Task Assignment:** Assigns Map and Reduce tasks to workers via RPC.
- **Monitoring & Fault Tolerance:** Monitors task progress and reassigns tasks if a worker fails (using a 10-second timeout).
- **Job Completion:** Provides a `Done()` method that signals when the entire job is complete.

### Worker Processes
- **Task Requesting:** In a loop, contacts the coordinator via RPC to request a task.
- **Task Execution:** Reads input files, applies the application's Map/Reduce functions, and writes output.
- **File Handling:** Writes intermediate files (for Map tasks) and final output files (for Reduce tasks).

## Implementation Overview

Inspired by the seminal MapReduce paper by Google, this project follows these core steps during execution:

1. **Input Splitting:**  
   The MapReduce library splits the input files into *M* pieces (splits), each typically between 16MB and 64MB.

2. **Task Assignment:**  
   One process acts as the master (coordinator) while the others function as workers.  
   The master assigns *M* map tasks and *R* reduce tasks to idle workers.

3. **Map Phase:**  
   A worker assigned a map task reads its corresponding input split, extracts key-value pairs, and applies the user-defined Map function.  
   Intermediate key-value pairs are buffered in memory.

4. **Intermediate Data Handling:**  
   Buffered data is periodically flushed to the local disk and partitioned into *R* regions using a partitioning function (e.g., `hash(key) mod R`).  
   The locations of these intermediate files are communicated back to the master.

5. **Reduce Phase:**  
   The master notifies reduce workers about the locations of the intermediate data.  
   Reduce workers fetch the data via RPC from the map workers’ local disks, then sort it by key to group together all values associated with the same key.

6. **Reduce Function Execution:**  
   For each unique key, the user-defined Reduce function is applied to the grouped values.  
   The results are written to a final output file corresponding to that reduce partition.

7. **Completion:**  
   Once all map and reduce tasks are complete, the master signals the end of the job.  
   The final output is available across *R* output files, which can be used for further processing if needed.

## Getting Started

### Prerequisites
- **Go:** Version 1.18 or later is recommended. You can download it from [golang.org](https://golang.org/dl/).
- **Git:** To clone the repository.

### Installation

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd mapreduce
   ```
2. **Initialize the Go module (if not already done):**
   ```bash
   go mod init mapreduce
   ```
3. **Tidy up dependencies:**
   ```bash
   go mod tidy
   ```

## Usage

### Running a MapReduce Job

0. **Build the shared object (Application code):**
   ```bash
   go build -buildmode=plugin mrapps/wc.go
   ```
1. **Start the Coordinator (or Master) with input `.txt` files:**
   ```bash
   go run main/mrcoordinator.go pg-*.txt
   ```
2. **Start one or more Workers & specify the application (wordcount - `wc.so`):**
   ```bash
   go run main/mrworker.go wc.so
   ```

## Project Structure

```
mapreduce/
├── main/          # Entry points (coordinator, worker)
│   ├── mrcoordinator.go
│   ├── mrworker.go
├── mr/            # Core MapReduce framework logic
│   ├── coordinator.go
│   ├── worker.go
│   ├── rpc.go
├── mrapps/        # Example applications demonstrating usage
├── go.mod         # Go module file
├── go.sum         # Dependency checksums
└── README.md      # Project documentation
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
