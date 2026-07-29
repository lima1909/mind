package index

import (
	"errors"
	"testing"

	"github.com/lima1909/mind/errr"
)

type TestObject struct {
	ID   uint32
	Name string
}

func TestIDSliceIndex_MainAndCornerCases(t *testing.T) {
	getIDField := func(obj *TestObject) uint32 { return obj.ID }

	t.Run("Main Case: Set and Get successfully", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)
		obj := &TestObject{ID: 10, Name: "Item 10"}

		idx.Set(obj, 500) // maps object ID 10 -> internal list index 500

		lidx, found := idx.GetIndex(10)
		if !found {
			t.Fatalf("expected found")
		}
		if lidx != 500 {
			t.Errorf("expected list index 500, got %d", lidx)
		}

		id := idx.GetID(obj)
		lidx2, found := idx.GetIndex(id)
		if !found || id != 10 || lidx2 != 500 {
			t.Errorf("GetID failed: got id=%d, lidx=%d", id, lidx2)
		}
	})

	t.Run("Corner Case: Slice growth within capacity limits", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)

		// Artificially allocate high capacity but low length
		idx.idToIdx = make([]uintBucket, 2, 100)

		obj := &TestObject{ID: 50, Name: "Inside Capacity Limit"}
		// This hits the 'else' block inside length checks: si.idToIdx[:id+1]
		idx.Set(obj, 12)

		lidx, found := idx.GetIndex(50)
		if !found || lidx != 12 {
			t.Errorf("failed to retrieve index after growing within capacity limits")
		}
	})

	t.Run("Corner Case: Slice growth exceeding capacity (Reallocation Trigger)", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)

		obj1 := &TestObject{ID: 2, Name: "First"}
		idx.Set(obj1, 100)

		// Set a massive ID to force reallocation and array copying
		obj2 := &TestObject{ID: 5000, Name: "Far Out"}
		idx.Set(obj2, 200)

		// Ensure old data survived reallocation copy phase
		lidx1, found := idx.GetIndex(2)
		if !found || lidx1 != 100 {
			t.Errorf("old data corrupted during capacity growth phase")
		}

		// Ensure new data was assigned correctly
		lidx2, found := idx.GetIndex(5000)
		if !found || lidx2 != 200 {
			t.Errorf("new data out of bounds assignment failed")
		}
	})

	t.Run("Corner Case: Querying unassigned index slots (Holes)", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)
		idx.Set(&TestObject{ID: 5, Name: "Five"}, 50)

		// Index 3 exists inside the backing array bounds but is unoccupied
		_, found := idx.GetIndex(3)
		if found {
			t.Error("expected error looking up empty data hole, got nil")
		}
	})

	t.Run("Corner Case: Out of Bounds Lookups", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)
		idx.Set(&TestObject{ID: 5, Name: "Five"}, 50)

		// Look up an ID completely beyond slice length
		_, found := idx.GetIndex(9999)
		if found {
			t.Error("expected error looking up out-of-bounds index, got nil")
		}
	})

	t.Run("Main Case: UnSet elements", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)
		obj := &TestObject{ID: 15, Name: "To Be Deleted"}

		idx.Set(obj, 80)
		idx.UnSet(obj, 80)

		_, found := idx.GetIndex(15)
		if found {
			t.Error("expected value to be cleared after calling UnSet, but it was found")
		}

		// Corner case: Unsetting an out-of-bounds item should safely return without panicking
		idx.UnSet(&TestObject{ID: 9999}, 0)
	})

	t.Run("Main Case: BulkSet Iteration Evaluation", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)

		items := []*TestObject{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
			{ID: 3, Name: "C"},
		}

		// Convert standard slice to Go 1.23+ Seq2 format
		seq := func(yield func(int, *TestObject) bool) {
			for i, item := range items {
				if !yield(i+10, item) { // maps IDs to virtual list positions 10, 11, 12
					return
				}
			}
		}

		idx.BulkSet(seq)

		// If the value receiver bug is active, this lookup will fail
		lidx, found := idx.GetIndex(3)
		if !found || lidx != 12 {
			t.Errorf("BulkSet failed to safely update data structure: found=%v, lidx=%d", found, lidx)
		}
	})

	t.Run("Main Case: Equal Method Type Safety", func(t *testing.T) {
		idx := NewIDSliceIndex(getIDField)
		idx.Set(&TestObject{ID: 7, Name: "Seven"}, 70)

		// Valid evaluation
		rawIDs, err := idx.Equal(uint32(7))
		if err != nil || rawIDs.Count() != 1 || !rawIDs.Contains(70) {
			t.Errorf("Equal evaluation failed: %v", err)
		}

		// Corner Case: Invalid type assertion argument passes
		_, err = idx.Equal("7") // string input instead of uint32
		if err == nil {
			t.Error("expected type error when passing invalid data type string to Equal()")
		}
		if !errors.As(err, &errr.InvalidValueTypeError[uint32]{}) {
			t.Errorf("expected InvalidValueTypeError, got %T", err)
		}
	})
}
