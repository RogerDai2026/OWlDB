package concurrentSkipList

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"testing"
)

func TestSkiplist_Query(t *testing.T) {
	sl := NewSL[int, string](1, 100)

	res, err := sl.Query(context.TODO(), 11, 65)

	if err != nil {
		print(res)
	}
}

func TestSkiplist_FindElemDoesntExist(t *testing.T) {
	sl := NewSL[int, string](1, 100)

	//sl.Insert(12, "test1")

	_, found := sl.Find(54)

	if found {
		t.Errorf("TestSkiplist_Find failed, expected found = false, got found = true")
	}
}

func TestSkiplist_FindElemExists(t *testing.T) {
	sl := NewSL[int, string](-1, 100)

	chk := func(int, string, bool) (string, error) {
		return "hello", nil
	}

	_, err := sl.Upsert(4, chk)

	if err != nil {
		t.Errorf("Upsert failed: reason... %s", err.Error())
	}

	foundVal, found := sl.Find(4)

	if !found {
		t.Errorf("TestSkiplist_FindElemExists failed, element was not found")
	}

	if foundVal != "hello" {
		t.Errorf("TestSkiplist_FindElem failed, expected found value to be \"hello\",got %s", foundVal)
	}
}

func TestSkiplist_Remove(t *testing.T) {
	chk := func(int, string, bool) (string, error) {
		return "hello", nil
	}
	sl := NewSL[int, string](-1, 100)
	_, err := sl.Upsert(4, chk)

	if err != nil {
		t.Errorf("TestSkiplist_Remove failed, upsert failed for reason %s", err.Error())
	}

	removedVal, removed := sl.Remove(4)

	if removed == false {
		t.Errorf("TestSkiplist_Remove failed, element was not removed")
	}

	if removedVal != "hello" {
		t.Errorf("TestSkiplist_Remove failed, element was not removed")
	}

}

func TestSkiplist_RemoveNotInList(t *testing.T) {
	sl := NewSL[int, string](-1, 100)
	_, removed := sl.Remove(5) //not in the list

	if removed {
		t.Errorf("TestSkiplist_RemoveNotInList failed, a nonexistent element was removed")
	}
}

func checkFactory[K cmp.Ordered, V any](newVal V) func(K, V, bool) (V, error) {

	return func(key K, curVal V, exists bool) (V, error) {

		//if exists {
		//	return nullV, fmt.Errorf("Error: already exists")
		//}
		return newVal, nil
	}
}

func checkFactoryNoUpsertIfErr[K cmp.Ordered, V any](newVal V) func(K, V, bool) (V, error) {
	return func(key K, curVal V, exists bool) (V, error) {

		if !exists {
			return curVal, fmt.Errorf("not existsing")
		}
		return newVal, nil
	}
}
func TestSkiplist_Insert2(t *testing.T) {
	sl := NewSL[int, string](1, 100)

	sl.Upsert(10, checkFactory[int, string]("1234"))

	foundVal, _ := sl.Find(10)
	sl.getState()
	fmt.Printf("We found %s the first time\n", foundVal)

	sl.Upsert(10, checkFactory[int, string]("ABC"))

	foundVal2, _ := sl.Find(10)

	fmt.Printf("We found %s the second time\n", foundVal2)

	sl.getState()

}

func upsertWorker[K int, V string](id int, wg *sync.WaitGroup, sl *Skiplist[K, V]) {
	defer wg.Done()
	chk := checkFactory[K, V]("value")

	_, err := sl.Upsert(K(id), chk)
	if err != nil {
		fmt.Printf(err.Error())
	}
}

func upsertWorker2[K int, V string](id int, wg *sync.WaitGroup, sl *Skiplist[K, V]) {
	defer wg.Done()
	chk := checkFactoryNoUpsertIfErr[K, V]("value")

	_, err := sl.Upsert(K(id), chk)
	if err != nil {
		slog.Debug(fmt.Sprintf("error"))
	}
}

func removeWorker[K int, V string](id int, wg *sync.WaitGroup, sl *Skiplist[K, V]) {
	defer wg.Done()
	removedVal, removed := sl.Remove(K(id))
	if removed {
		slog.Debug(fmt.Sprintf("Removed key %d, its associated value is %+v", id, removedVal))
	}
}

// TestConcurrentUpsert will upsert in multiple goroutines, ensuring that no deadlocks occur
func TestConcurrentUpsert(t *testing.T) {
	//logOpts := &logger.PrettyHandlerOptions{
	//	Level:    slog.LevelDebug, // log level you want
	//	Colorize: true,            // true or false
	//}

	//handler := logger.NewPrettyHandler(os.Stdout, logOpts)
	//logger := slog.New(handler)
	//slog.SetDefault(logger)
	var wg sync.WaitGroup
	sl := NewSL[int, string](-1, 1000)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go upsertWorker[int, string](i, &wg, sl)
	}
	wg.Wait()
	sl.getState()
}

func TestSequentialLargeOperations(t *testing.T) {
	numElements := 1000 // Size of the test

	sl := NewSL[int, string](-1, 100000)

	// Insert a large number of elements
	for i := 1; i <= numElements; i++ {
		sl.Upsert(i, checkFactory[int, string](strconv.FormatInt(int64(i), 10)))
	}

	// Verify that all elements are inserted correctly
	for i := 1; i <= numElements; i++ {
		val, found := sl.Find(i)
		expectedVal := fmt.Sprintf("%d", i)
		if !found || val != expectedVal {
			t.Fatalf("Expected to find key %d with value %s, but got %v", i, expectedVal, val)
		}
	}

	// Remove half of the elements
	for i := 1; i <= numElements/2; i++ {
		_, removed := sl.Remove(i)
		if !removed {
			t.Fatalf("Failed to remove key %d", i)
		}
	}

	// Verify that removed elements are not found
	for i := 1; i <= numElements/2; i++ {
		_, found := sl.Find(i)
		if found {
			t.Fatalf("Key %d was found after it was removed", i)
		}
	}

	// Verify that the remaining elements still exist
	for i := numElements/2 + 1; i <= numElements; i++ {
		val, found := sl.Find(i)
		expectedVal := fmt.Sprintf("%d", i)
		if !found || val != expectedVal {
			t.Fatalf("Expected to find key %d with value %s, but got %v", i, expectedVal, val)
		}
	}

	// Perform a query for the remaining elements
	ctx := context.TODO()
	_, err := sl.Query(ctx, numElements/2+1, numElements)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Large sequential test with %d elements passed successfully.", numElements)
}

//func upsertWorker[K int, V string](id int, wg *sync.WaitGroup, sl *Skiplist[K, V]) {
//	defer wg.Done()
//	chk := checkFactory[K, V]("value")
//
//	_, err := sl.Upsert(K(id), chk)
//	if err != nil {
//		fmt.Printf(err.Error())
//	}
//}

func findWorker(key int, wg *sync.WaitGroup, sl *Skiplist[int, string]) {
	defer wg.Done()

	sl.Find(key)
}

// please don't deadlock
func TestUpsertRemove(t *testing.T) {

	var wg sync.WaitGroup
	//please don't deadlock
	sl := NewSL[int, string](-5, 200000000)
	wg.Add(97 * 2)
	for i := 3; i < 100; i++ { //for more stress, set this higher

		go upsertWorker[int, string](i, &wg, sl)

		go removeWorker[int, string](i, &wg, sl)

	}
	wg.Wait()
	sl.getState()
	fmt.Printf("\n")

}

// tests two consecutive upserts
func TestDoubleUpsert(t *testing.T) {
	var wg sync.WaitGroup
	//please don't deadlock
	sl := NewSL[int, string](-5, 2000)

	for i := 1; i <= 10; i++ { //for more stress, set this higher
		wg.Add(2)
		go upsertWorker[int, string](i, &wg, sl)
		go upsertWorker[int, string](i, &wg, sl)
	}
	wg.Wait()
}

func TestConcurrentUpdateDelete(t *testing.T) {
	var wg sync.WaitGroup

	sl := NewSL[int, string](-5, 2000000)
	for i := 3; i < 1000; i++ { //for more stress, set this higher

		wg.Add(3)
		go upsertWorker[int, string](i, &wg, sl)
		go upsertWorker[int, string](i, &wg, sl)
		go findWorker(i, &wg, sl)
	}
	wg.Wait()
}

func TestUpsertFailIfNotExist(t *testing.T) {
	sl := NewSL[int, string](-1, 200)
	sl.Upsert(1, checkFactory[int, string]("abc"))
	sl.Upsert(2, checkFactory[int, string]("abc"))
	sl.Upsert(3, checkFactory[int, string]("abc"))

	_, err := sl.Upsert(4, checkFactoryNoUpsertIfErr[int, string]("val"))
	if err == nil {
		t.Errorf("TestUpsertFailIfNotExist, err is nil")
	}
}
