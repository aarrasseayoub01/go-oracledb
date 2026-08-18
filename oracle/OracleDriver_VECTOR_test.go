package oracle

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

// TestDriver_Vector_Basic verifies INSERT, SELECT, UPDATE, and DELETE for all
// dense VECTOR element types supported by the driver.
func TestDriver_Vector_Basic(t *testing.T) {
	if TestingConfig == nil {
		t.Skip("No configuration available")
	}
	if TestingConfig.DatabaseVersion.Major < 23 {
		t.Skip("VECTOR datatype requires Oracle Database 23c+")
	}

	ctx := context.Background()
	const dimensions = 1024
	values := make(VectorFloat64, dimensions)
	updatedValues := make(VectorFloat64, dimensions)
	for i := range values {
		values[i] = float64(i) / 4
		updatedValues[i] = -float64(i) / 8
	}

	tests := []struct {
		name        string
		table       string
		definition  string
		insert      any
		want        any
		update      any
		updatedWant any
	}{
		{"float64", "vector_f64", "VECTOR(1024, FLOAT64)", values, []float64(values), updatedValues, []float64(updatedValues)},
		{"float32", "vector_f32", "VECTOR(3, FLOAT32)", VectorFloat32{4.5, -5.25, 6.75}, []float32{4.5, -5.25, 6.75}, VectorFloat32{-1.5, 2.25, 9.75}, []float32{-1.5, 2.25, 9.75}},
		{"int8", "vector_i8", "VECTOR(3, INT8)", VectorInt8{1, -2, 3}, []int8{1, -2, 3}, VectorInt8{-8, 0, 7}, []int8{-8, 0, 7}},
		{"binary", "vector_binary", "VECTOR(16, BINARY)", VectorBinary{0xAA, 0x0F}, []byte{0xAA, 0x0F}, VectorBinary{0x55, 0xF0}, []byte{0x55, 0xF0}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := openTestDBWithConfig(TestingConfig)
			if err != nil {
				t.Fatalf("open test database: %v", err)
			}
			defer db.Close()

			_ = dropTable(ctx, db, tc.table)
			if err := createTable(ctx, db, tc.table, map[string]string{"id": "NUMBER PRIMARY KEY", "vec": tc.definition}); err != nil {
				t.Fatalf("create table: %v", err)
			}
			defer func() { _ = dropTable(ctx, db, tc.table) }()

			if result, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, vec) VALUES (:1, :2)", tc.table), 1, tc.insert); err != nil {
				t.Fatalf("insert: %v", err)
			} else if count, err := result.RowsAffected(); err != nil || count != 1 {
				t.Fatalf("insert rows affected: got %d, err=%v", count, err)
			}

			assertVector := func(want any) {
				var got any
				switch want.(type) {
				case []float64:
					got = new([]float64)
				case []float32:
					got = new([]float32)
				case []int8:
					got = new([]int8)
				case []byte:
					got = new([]byte)
				}
				if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT vec FROM %s WHERE id = :1", tc.table), 1).Scan(got); err != nil {
					t.Fatalf("fetch: %v", err)
				}
				if !reflect.DeepEqual(reflect.ValueOf(got).Elem().Interface(), want) {
					t.Fatalf("vector mismatch: got %v want %v", reflect.ValueOf(got).Elem().Interface(), want)
				}
			}
			assertVector(tc.want)

			if result, err := db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET vec = :1 WHERE id = :2", tc.table), tc.update, 1); err != nil {
				t.Fatalf("update: %v", err)
			} else if count, err := result.RowsAffected(); err != nil || count != 1 {
				t.Fatalf("update rows affected: got %d, err=%v", count, err)
			}
			assertVector(tc.updatedWant)

			if result, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = :1", tc.table), 1); err != nil {
				t.Fatalf("delete: %v", err)
			} else if count, err := result.RowsAffected(); err != nil || count != 1 {
				t.Fatalf("delete rows affected: got %d, err=%v", count, err)
			}
		})
	}
}

// TestDriver_SparseVector_Basic exercises sparse VECTOR bind and fetch support
// for every element type. It uses zero-based indices, matching the database
// representation, and verifies the complete INSERT, UPDATE, and DELETE flow.
func TestDriver_SparseVector_Basic(t *testing.T) {
	if TestingConfig == nil {
		t.Skip("No configuration available")
	}
	if TestingConfig.DatabaseVersion.Major < 23 {
		t.Skip("VECTOR datatype requires Oracle Database 23c+")
	}

	ctx := context.Background()
	tests := []struct {
		name        string
		tableName   string
		vecDDL      string
		bind        any
		updatedBind any
		newDest     func() any
		want        any
		updatedWant any
	}{
		{
			name:        "float64",
			tableName:   "sparse_vector_f64",
			vecDDL:      "VECTOR(8, FLOAT64, SPARSE)",
			bind:        SparseVectorFloat64{Dimensions: 8, Indices: []uint32{0, 3, 7}, Values: []float64{1.25, -2.5, 3.75}},
			updatedBind: SparseVectorFloat64{Dimensions: 8, Indices: []uint32{1, 6}, Values: []float64{-4.5, 5.25}},
			newDest:     func() any { return &SparseVectorFloat64{} },
			want:        SparseVectorFloat64{Dimensions: 8, Indices: []uint32{0, 3, 7}, Values: []float64{1.25, -2.5, 3.75}},
			updatedWant: SparseVectorFloat64{Dimensions: 8, Indices: []uint32{1, 6}, Values: []float64{-4.5, 5.25}},
		},
		{
			name:        "float32",
			tableName:   "sparse_vector_f32",
			vecDDL:      "VECTOR(8, FLOAT32, SPARSE)",
			bind:        SparseVectorFloat32{Dimensions: 8, Indices: []uint32{1, 5}, Values: []float32{2.5, -3.25}},
			updatedBind: SparseVectorFloat32{Dimensions: 8, Indices: []uint32{2, 4}, Values: []float32{-1.5, 6.75}},
			newDest:     func() any { return &SparseVectorFloat32{} },
			want:        SparseVectorFloat32{Dimensions: 8, Indices: []uint32{1, 5}, Values: []float32{2.5, -3.25}},
			updatedWant: SparseVectorFloat32{Dimensions: 8, Indices: []uint32{2, 4}, Values: []float32{-1.5, 6.75}},
		},
		{
			name:        "int8",
			tableName:   "sparse_vector_i8",
			vecDDL:      "VECTOR(8, INT8, SPARSE)",
			bind:        SparseVectorInt8{Dimensions: 8, Indices: []uint32{2, 6}, Values: []int8{-2, 7}},
			updatedBind: SparseVectorInt8{Dimensions: 8, Indices: []uint32{0, 7}, Values: []int8{4, -8}},
			newDest:     func() any { return &SparseVectorInt8{} },
			want:        SparseVectorInt8{Dimensions: 8, Indices: []uint32{2, 6}, Values: []int8{-2, 7}},
			updatedWant: SparseVectorInt8{Dimensions: 8, Indices: []uint32{0, 7}, Values: []int8{4, -8}},
		},
		{
			name:        "binary",
			tableName:   "sparse_vector_bin",
			vecDDL:      "VECTOR(8, BINARY, SPARSE)",
			bind:        SparseVectorBinary{Dimensions: 8, Indices: []uint32{0, 4, 7}},
			updatedBind: SparseVectorBinary{Dimensions: 8, Indices: []uint32{2, 3}},
			newDest:     func() any { return &SparseVectorBinary{} },
			want:        SparseVectorBinary{Dimensions: 8, Indices: []uint32{0, 4, 7}},
			updatedWant: SparseVectorBinary{Dimensions: 8, Indices: []uint32{2, 3}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := openTestDBWithConfig(TestingConfig)
			if err != nil {
				t.Fatalf("open test database: %v", err)
			}
			defer db.Close()

			_ = dropTable(ctx, db, tc.tableName)
			if err := createTable(ctx, db, tc.tableName, map[string]string{
				"id":  "NUMBER PRIMARY KEY",
				"vec": tc.vecDDL,
			}); err != nil {
				t.Fatalf("create table: %v", err)
			}
			defer func() { _ = dropTable(ctx, db, tc.tableName) }()

			if _, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (id, vec) VALUES (:1, :2)", tc.tableName), 1, tc.bind); err != nil {
				t.Fatalf("insert sparse vector: %v", err)
			}
			assertFetched := func(phase string, want any) {
				got := tc.newDest()
				if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT vec FROM %s WHERE id = :1", tc.tableName), 1).Scan(got); err != nil {
					t.Fatalf("fetch %s: %v", phase, err)
				}
				if !reflect.DeepEqual(reflect.ValueOf(got).Elem().Interface(), want) {
					t.Fatalf("%s mismatch: got %#v want %#v", phase, got, want)
				}
			}
			assertFetched("after insert", tc.want)

			if _, err := db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET vec = :1 WHERE id = :2", tc.tableName), tc.updatedBind, 1); err != nil {
				t.Fatalf("update sparse vector: %v", err)
			}
			assertFetched("after update", tc.updatedWant)

			if _, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = :1", tc.tableName), 1); err != nil {
				t.Fatalf("delete sparse vector: %v", err)
			}
		})
	}
}
