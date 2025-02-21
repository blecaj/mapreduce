package mr

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

// Possible states for tasks
type TaskState int

const (
	TaskUnassigned TaskState = iota
	TaskInProgress
	TaskCompleted
)

// Task represents a Map or Reduce task.
type Task struct {
	ID        int       // Unique identifier for the task type.
	Type      string    // "map" or "reduce"
	FileName  string    // For map tasks, the input file name.
	State     TaskState // Current state: unassigned, in-progress, or completed.
	StartTime time.Time // The time when the task was assigned.
}
type Coordinator struct {
	// Your definitions here.
	mu          sync.Mutex // Ensures safe concurrent access
	mapTasks    []Task     // List of map tasks
	reduceTasks []Task     // List of reduce tasks
	nReduce     int        // Number of reduce tasks
	//taskQueue   chan *Task // Channel to distribute tasks
	done bool // Flag indicating completion
}

// Coordinator’s RPC handler
// Assigning a Task to a Worker. RPC method (1)
// Coordinator listens, to see if worker is making call to it for a new task.
func (c *Coordinator) GetTask(args *ExampleArgs, reply *TaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.mapTasks {
		if c.mapTasks[i].State == TaskUnassigned {
			c.mapTasks[i].State = TaskInProgress
			c.mapTasks[i].StartTime = time.Now()
			reply.Task = c.mapTasks[i] //// Assign the task to the reply
			return nil
		}
	}
	// Assign Reduce tasks only after the final Map task is completed
	for i := range c.reduceTasks {
		if c.reduceTasks[i].State == TaskUnassigned && c.mapTasks[len(c.mapTasks)-1].State == TaskCompleted {
			c.reduceTasks[i].State = TaskInProgress
			c.reduceTasks[i].StartTime = time.Now()
			reply.Task = c.reduceTasks[i] //// Assign the task to the reply
			return nil
		}
	}
	return nil
}

// Handling Task Completion and Timeouts RPC handler.
// An RPC method (2) for workers to report task completion
// When a worker reports that it has completed a task, update the task’s state:
func (c *Coordinator) MarkTaskCompleted(args *MarkTaskCompletedArgs, reply *MarkTaskCompletedReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Mark the task as completed based on its type.
	if args.TaskType == "map" {
		if args.TaskID < len(c.mapTasks) {
			c.mapTasks[args.TaskID].State = TaskCompleted
		} else {
			return fmt.Errorf("invalid map task ID: %v", args.TaskID)
		}
	} else if args.TaskType == "reduce" {
		if args.TaskID < len(c.reduceTasks) {
			c.reduceTasks[args.TaskID].State = TaskCompleted
		} else {
			return fmt.Errorf("invalid reduce task ID: %v", args.TaskID)
		}
	} else {
		return fmt.Errorf("unknown task type: %v", args.TaskType)
	}

	reply.Success = true
	fmt.Printf("filename is %v\n", reply.Success)
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.nReduce = nReduce

	// Initialize mapTask based on input Files
	for i, file := range files {
		task := Task{
			FileName: file,
			ID:       i,
			Type:     "map",
			State:    TaskUnassigned,
		}

		c.mapTasks = append(c.mapTasks, task)
		// Initialize ReduceTasks based on nReduce and File i
		for j := 0; j < c.nReduce; j++ {
			reduceTask := Task{
				FileName: "mr-" + strconv.Itoa(i) + "-" + strconv.Itoa(j),
				ID:       j,
				Type:     "reduce",
				State:    TaskUnassigned,
			}
			c.reduceTasks = append(c.reduceTasks, reduceTask)
		}
	}
	//fmt.Printf("filename is %v\n", c.mapTasks[0].FileName)
	//Task ID: 0, Type: map, File: file1.txt, State: Unassigned
	//Task ID: 0, Type: reduce, File: mr-0-0, State: Unassigned
	//Task ID: 1, Type: reduce, File: mr-0-1, State: Unassigned
	//Task ID: 2, Type: reduce, File: mr-0-2, State: Unassigned

	c.server()

	return &c
}
