package task_test

import (
	"context"
	"errors"
	"fmt"

	"example.com/go-cloud-native-study/internal/task"
)

func ExampleMemoryStore_zeroValue() {
	service := task.NewService(&task.MemoryStore{})

	created, err := service.Create(context.Background(), "  learn boundaries  ")
	if err != nil {
		fmt.Println("create:", err)
		return
	}

	got, err := service.Get(context.Background(), created.ID)
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	fmt.Println(got.ID, got.Title)
	// Output: 1 learn boundaries
}

func ExampleService_Get_errorContract() {
	service := task.NewService(&task.MemoryStore{})

	_, err := service.Get(context.Background(), "missing")
	fmt.Println(errors.Is(err, task.ErrNotFound))
	fmt.Println(err)
	// Output:
	// true
	// get task "missing": task not found
}
