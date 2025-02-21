package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// buckets function partitions key-value pairs into nReduce buckets
func buckets(nReduce int, kvs []KeyValue) []map[string][]string {
	// Step 1: Create a slice of maps
	buckets := make([]map[string][]string, nReduce)
	for i := range buckets {
		buckets[i] = make(map[string][]string) // Initialize each map
	}

	// Step 2: Distribute key-value pairs into nReduce buckets
	for _, kv := range kvs {
		bucketIndex := ihash(kv.Key) % nReduce // Determine the bucket
		buckets[bucketIndex][kv.Key] = append(buckets[bucketIndex][kv.Key], kv.Value)
	}
	return buckets
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	nReduce := 10
	for {
		// Request a new task from the coordinator.
		reply, err := CallGetTasks()
		if err != nil {
			log.Fatalf("RPC call failed: %v", err)
		}

		// If no task is available, sleep briefly and try again.
		// (This assumes an empty FileName indicates no task.)
		if reply.Task.FileName == "" {
			time.Sleep(time.Second)
			continue
		}

		// ------MAP WORKER ----------
		if reply.Task.Type == "map" {
			file, err := os.Open(reply.Task.FileName)
			if err != nil {
				log.Fatalf("cannot open %v", reply.Task.FileName)
			}
			defer file.Close()

			// Read file content
			content, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", reply.Task.FileName)
			}

			KeyValue := mapf(reply.Task.FileName, string(content))
			intermediateBuckets := buckets(nReduce, KeyValue)

			for i, bucket := range intermediateBuckets {
				oname := fmt.Sprintf("mr-%d-%d", reply.Task.ID, i)

				ofile, err := os.Create(oname)
				if err != nil {
					log.Fatalf("Failed to create file %s: %v", oname, err)
				}

				enc := json.NewEncoder(ofile)
				for _, kv := range bucket {
					if err := enc.Encode(&kv); err != nil {
						log.Fatalf("Failed to encode key-value pair %v: %v", kv, err)
					}
				}
				ofile.Close() // Explicitly close instead of defer
			}
		} else if reply.Task.Type == "reduce" {
			// ------REDUCE WORKER ----------
			fmt.Println(reply.Task.ID, "is a reduce task")

			var kvs []KeyValue

			// Read all intermediate files (mr-X-Y) for this reduce task
			for i := 0; i < nReduce; i++ {
				inFileName := fmt.Sprintf("mr-%d-%d", i, reply.Task.ID)
				inFile, err := os.Open(inFileName)
				if err != nil {
					log.Printf("Warning: Could not open intermediate file %s: %v", inFileName, err)
					continue // Skip missing files instead of crashing
				}

				dec := json.NewDecoder(inFile)
				for {
					var kv KeyValue
					if err := dec.Decode(&kv); err != nil {
						break // Exit loop when decoding fails (EOF)
					}
					kvs = append(kvs, kv)
				}
				inFile.Close() // Close file after reading
			}

			// Aggregate values by key
			results := make(map[string][]string)
			for _, kv := range kvs {
				results[kv.Key] = append(results[kv.Key], kv.Value)
			}

			// Write reduced results to output file
			outputFileName := fmt.Sprintf("mr-out-%d", reply.Task.ID)
			outputFile, err := os.Create(outputFileName)
			if err != nil {
				log.Fatalf("Failed to create output file %s: %v", outputFileName, err)
			}
			defer outputFile.Close()

			for key, values := range results {
				result := reducef(key, values)
				fmt.Fprintf(outputFile, "%s %s\n", key, result) // Corrected fmt.Fprintf usage
			}
		}
		//Mark the task as completed.
		callCompletion(reply.Task.ID, reply.Task.Type, true)
	}
}

// ***RPC setup & Calls***

// function to show how to make an RPC call (1) to the coordinator.
// the RPC argument and reply types are defined in rpc.go.
// This function calls GetTask() method from struct Coordinator, from coordinator.go program
// It will return the response it recieives from it or an error.
func CallGetTasks() (TaskReply, error) {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := TaskReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the GetTask() method of struct Coordinator.
	ok := call("Coordinator.GetTask", &args, &reply)
	if ok {
		// reply.Y should be 100.
		return reply, nil
	} else {
		return reply, fmt.Errorf("RPC call failed")
	}
}

// Function to make RPC (2) to coordinator to tell it when task is completed
// The expected reply is MarkTaskCompletedArgs Struct {TaskID int TaskType string}, MarkTaskCompletedReply {bool}
func callCompletion(taskid int, tasktype string, complete bool) bool {
	// Construct the arguments and reply using field names
	args := MarkTaskCompletedArgs{
		TaskID:   taskid,
		TaskType: tasktype,
	}
	reply := MarkTaskCompletedReply{
		Success: complete,
	}
	return call("Coordinator.MarkTaskCompleted", &args, &reply)
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
